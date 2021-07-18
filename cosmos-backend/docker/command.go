package docker

import (
	"bufio"
	"bytes"
	"context"
	"cosmos"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigDefault

var _ cosmos.CommandService = (*CommandService)(nil)

type CommandService struct {
	*cosmos.App
}

func NewCommandService() *CommandService {
	return &CommandService{}
}

const (
	NormalizationDockerImage = "airbyte/normalization:0.1.36"
)

func recoverFromPanic() {
	if err := recover(); err != nil {
		log.Printf("worker panic: %s", err)
		debug.PrintStack()
	}
}

func (s *CommandService) Spec(ctx context.Context, connector *cosmos.Connector) (*cosmos.Message, error) {
	return s.Runner(ctx, connector, nil, cosmos.MessageTypeSpec)
}

func (s *CommandService) Check(ctx context.Context, connector *cosmos.Connector, config interface{}) (*cosmos.Message, error) {
	return s.Runner(ctx, connector, config, cosmos.MessageTypeConnectionStatus)
}

func (s *CommandService) Discover(ctx context.Context, connector *cosmos.Connector, config interface{}) (*cosmos.Message, error) {
	// Destination connectors don't support "discover".
	if connector.Type == cosmos.ConnectorTypeDestination {
		return &cosmos.Message{Type: cosmos.MessageTypeCatalog}, nil
	}
	return s.Runner(ctx, connector, config, cosmos.MessageTypeCatalog)
}

func (s *CommandService) Read(ctx context.Context, connector *cosmos.Connector, empty bool) (<-chan interface{}, <-chan error) {
	out := make(chan interface{}, 100)
	errc := make(chan error, 1)

	// empty is true when we want to wipe the destination clean.
	// Don't send any data from source in this case.
	if empty {
		close(out)
		errc <- nil
		return out, errc
	}

	go func() {
		defer recoverFromPanic()
		defer close(out)

		artifactory := cosmos.ArtifactoryFromContext(ctx)
		configFile := s.App.GetArtifactPath(artifactory, cosmos.ArtifactSrcConfig)
		configuredCatalogFile := s.App.GetArtifactPath(artifactory, cosmos.ArtifactCatalog)
		stateFile := s.App.GetArtifactPath(artifactory, cosmos.ArtifactBeforeState)

		dockerImage := connector.DockerImageName + ":" + connector.DockerImageTag
		cmdString := prepareDockerCmd("read", false, dockerImage, nil, configFile, configuredCatalogFile, stateFile)
		s.sendOutput(ctx, out, fmt.Sprintf("Docker command: docker %s", cmdString))

		cmd := exec.CommandContext(ctx, "docker", strings.Split(cmdString, " ")...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errc <- fmt.Errorf("failed to get stdout pipe in read command. err: %w", err)
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			errc <- fmt.Errorf("failed to get stderr pipe in read command. err: %w", err)
			return
		}
		if err := cmd.Start(); err != nil {
			errc <- fmt.Errorf("failed to start read command. err: %w", err)
			return
		}

		s.scanOutput(ctx, stdout, stderr, out)

		if err := cmd.Wait(); err != nil {
			errc <- fmt.Errorf("read command failed with err: %w", err)
			return
		}

		errc <- nil
	}()

	return out, errc
}

func (s *CommandService) Write(ctx context.Context, connector *cosmos.Connector, in <-chan *cosmos.Message) (<-chan interface{}, <-chan error) {
	out := make(chan interface{}, 100)
	errc := make(chan error, 1)

	go func() {
		defer recoverFromPanic()
		defer close(out)

		var wg sync.WaitGroup

		artifactory := cosmos.ArtifactoryFromContext(ctx)
		configFile := s.App.GetArtifactPath(artifactory, cosmos.ArtifactDstConfig)
		configuredCatalogFile := s.App.GetArtifactPath(artifactory, cosmos.ArtifactCatalog)

		dockerImage := connector.DockerImageName + ":" + connector.DockerImageTag
		cmdString := prepareDockerCmd("write", true, dockerImage, nil, configFile, configuredCatalogFile, nil)
		s.sendOutput(ctx, out, fmt.Sprintf("Docker command: docker %s", cmdString))

		cmd := exec.CommandContext(ctx, "docker", strings.Split(cmdString, " ")...)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			errc <- fmt.Errorf("failed to get stdin pipe in write command. err: %w", err)
			return
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errc <- fmt.Errorf("failed to get stdout pipe in write command. err: %w", err)
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			errc <- fmt.Errorf("failed to get stderr pipe in write command. err: %w", err)
			return
		}
		if err := cmd.Start(); err != nil {
			errc <- fmt.Errorf("failed to start write command. err: %w", err)
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer stdin.Close()

			encoder := json.NewEncoder(stdin)
			for msg := range in {
				if msg.Type == cosmos.MessageTypeRecord {
					if err := encoder.Encode(msg); errors.Is(err, syscall.EPIPE) {
						break
					} else if err != nil {
						log.Printf("failed to encode message in write command. err: %s", err)
					}
				} else {
					// Destinations currently don't process state messages. So, just pass them through to the output.
					s.sendOutput(ctx, out, msg)
				}
			}
		}()

		s.scanOutput(ctx, stdout, stderr, out)
		wg.Wait()

		if err := cmd.Wait(); err != nil {
			errc <- fmt.Errorf("write command failed. err: %w", err)
			return
		}

		errc <- nil
	}()

	return out, errc
}

func (s *CommandService) Normalize(ctx context.Context, connector *cosmos.Connector, config map[string]interface{}) (<-chan interface{}, <-chan error) {
	out := make(chan interface{}, 100)
	errc := make(chan error, 1)

	go func() {
		defer recoverFromPanic()
		defer close(out)

		// check whether normalization has to be performed.
		skipNormalization := true
		if v, ok := config["basic_normalization"]; ok {
			if normalize, ok := v.(bool); ok && normalize {
				skipNormalization = false
			}
		}
		if skipNormalization {
			s.sendOutput(ctx, out, "Normalization is not available or is disabled. Skipping.")
			errc <- nil
			return
		}

		artifactory := cosmos.ArtifactoryFromContext(ctx)
		configFile := s.App.GetArtifactPath(artifactory, cosmos.ArtifactDstConfig)
		configuredCatalogFile := s.App.GetArtifactPath(artifactory, cosmos.ArtifactCatalog)

		cmdString := prepareDockerCmd("run", false, NormalizationDockerImage, &connector.DestinationType, configFile, configuredCatalogFile, nil)
		s.sendOutput(ctx, out, fmt.Sprintf("Docker command: docker %s", cmdString))

		cmd := exec.CommandContext(ctx, "docker", strings.Split(cmdString, " ")...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errc <- fmt.Errorf("failed to get stdout pipe in normalization command. err: %w", err)
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			errc <- fmt.Errorf("failed to get stderr pipe in normalization command. err: %w", err)
			return
		}
		if err := cmd.Start(); err != nil {
			errc <- fmt.Errorf("failed to start normalization command. err: %w", err)
			return
		}

		s.scanOutput(ctx, stdout, stderr, out)

		if err := cmd.Wait(); err != nil {
			errc <- fmt.Errorf("normalization command failed with err: %w", err)
			return
		}

		errc <- nil
	}()

	return out, errc
}

func (s *CommandService) Runner(
	ctx context.Context,
	connector *cosmos.Connector,
	config interface{},
	messageType string,
) (*cosmos.Message, error) {

	var configFile *os.File
	var configFileName string
	var err error

	if config != nil {
		configFile, err = getTempFile(config)
		if err != nil {
			return nil, err
		}
		defer os.Remove(configFile.Name())
		configFileName = configFile.Name()

		// For Docker-in-Docker, we have to return the path as it would be on the host.
		configFileName = strings.TrimPrefix(configFileName, cosmos.ScratchSpace)
		configFileName = filepath.Join(os.Getenv("SCRATCH_SPACE"), configFileName)
	}

	dockerImage := connector.DockerImageName + ":" + connector.DockerImageTag
	var cmd string

	switch messageType {
	case cosmos.MessageTypeSpec:
		cmd = prepareDockerCmd("spec", false, dockerImage, nil, nil, nil, nil)
	case cosmos.MessageTypeConnectionStatus:
		cmd = prepareDockerCmd("check", false, dockerImage, nil, &configFileName, nil, nil)
	case cosmos.MessageTypeCatalog:
		cmd = prepareDockerCmd("discover", false, dockerImage, nil, &configFileName, nil, nil)
	default:
		panic("Unhandled message type in docker runner")
	}

	out, err := exec.CommandContext(ctx, "docker", strings.Split(cmd, " ")...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %s command on docker image %s err=%w", messageType, dockerImage, err)
	}

	for _, row := range bytes.Split(out, []byte("\n")) {
		msg, err := s.App.CreateMessage(ctx, row)
		if err == nil && msg.Type == messageType {
			return msg, nil
		}
	}

	return nil, fmt.Errorf("docker runner failed to find any %s messages", messageType)
}

func (s *CommandService) scanOutput(ctx context.Context, stdout io.ReadCloser, stderr io.ReadCloser, out chan<- interface{}) {
	merged := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(merged)

	for scanner.Scan() {
		b := scanner.Bytes()
		var msg interface{}
		msg, err := s.App.CreateMessage(ctx, b)
		if err != nil {
			msg = string(b)
		}
		select {
		case out <- msg:
		case <-ctx.Done():
			return
		}
	}
}

func (s *CommandService) sendOutput(ctx context.Context, out chan<- interface{}, i interface{}) {
	select {
	case out <- i:
	case <-ctx.Done():
		break
	}
}

func getTempFile(contents interface{}) (tmpFile *os.File, err error) {
	defer func() {
		if err != nil && tmpFile != nil {
			os.Remove(tmpFile.Name())
			tmpFile = nil
		}
	}()

	tmpFile, err = ioutil.TempFile(cosmos.ScratchSpace, "cosmos-")
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(contents)
	if err != nil {
		return tmpFile, err
	}

	if _, err = tmpFile.Write(b); err != nil {
		return tmpFile, err
	}

	return tmpFile, nil
}

func prepareDockerCmd(
	cmd string,
	interactive bool,
	dockerImage string,
	destinationType *string,
	configFile *string,
	configuredCatalogFile *string,
	stateFile *string,
) string {

	builder := strings.Builder{}

	// using --mount syntax because docker doesn't handle paths with ':'.
	// See https://github.com/moby/moby/issues/8604#issuecomment-332673783
	volMount := "--mount type=bind,source=%s,destination=%s "

	builder.WriteString("run --rm --net host ")

	if interactive {
		builder.WriteString("-i ")
	}
	if configFile != nil {
		builder.WriteString(fmt.Sprintf(volMount, *configFile, "/tmp/cosmos-config"))
	}
	if configuredCatalogFile != nil {
		builder.WriteString(fmt.Sprintf(volMount, *configuredCatalogFile, "/tmp/cosmos-configured-catalog"))
	}
	if stateFile != nil {
		builder.WriteString(fmt.Sprintf(volMount, *stateFile, "/tmp/cosmos-state"))
	}

	builder.WriteString(fmt.Sprintf(volMount, os.Getenv("LOCAL_DIR"), "/local"))

	builder.WriteString(fmt.Sprintf("%s %s ", dockerImage, cmd))

	addConfig, addCatalog, addState, addIntegrationType := false, false, false, false

	switch cmd {
	case "spec":
	case "check", "discover":
		addConfig = true
	case "read":
		addConfig, addCatalog, addState = true, true, stateFile != nil
	case "write":
		addConfig, addCatalog = true, true
	case "run":
		addConfig, addCatalog, addIntegrationType = true, true, true
	}

	if addConfig {
		builder.WriteString("--config /tmp/cosmos-config ")
	}
	if addCatalog {
		builder.WriteString("--catalog /tmp/cosmos-configured-catalog ")
	}
	if addState {
		builder.WriteString("--state /tmp/cosmos-state ")
	}
	if addIntegrationType {
		builder.WriteString(fmt.Sprintf("--integration-type %s ", *destinationType))
	}

	return strings.TrimSpace(builder.String())
}
