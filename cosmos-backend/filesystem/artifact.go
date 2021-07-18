package filesystem

import (
	"cosmos"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigDefault

var _ cosmos.ArtifactService = (*ArtifactService)(nil)

type ArtifactService struct {
}

func NewArtifactService() *ArtifactService {
	return &ArtifactService{}
}

func (s *ArtifactService) GetArtifactory(syncID int, executionDate time.Time) (*cosmos.Artifactory, error) {
	path := filepath.Join(
		cosmos.ArtifactDir,
		strconv.Itoa(syncID),
		executionDate.Format(time.RFC3339),
	)

	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, err
	}

	return &cosmos.Artifactory{Path: path}, nil
}

func (s *ArtifactService) GetArtifactRef(artifactory *cosmos.Artifactory, id int, attempt int32) (*log.Logger, error) {
	var err error

	artifactory.Once[id].Do(func() {
		var file *os.File
		file, err = os.OpenFile(filepath.Join(artifactory.Path, cosmos.ArtifactNames[id]), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return
		}
		artifactory.Artifacts[id] = log.New(file, fmt.Sprintf("[Attempt %3d] ", attempt), log.LstdFlags)
	})

	if err != nil {
		return nil, err
	}

	if artifactory.Artifacts[id] == nil {
		return nil, fmt.Errorf("artifact for %s is unavailable", cosmos.ArtifactNames[id])
	}

	return artifactory.Artifacts[id], nil
}

func (s *ArtifactService) WriteArtifact(artifactory *cosmos.Artifactory, id int, contents interface{}) error {
	if reflect.ValueOf(contents).IsNil() {
		return nil
	}

	file, err := os.Create(filepath.Join(artifactory.Path, cosmos.ArtifactNames[id]))
	if err != nil {
		return err
	}
	defer file.Close()

	b, err := json.Marshal(contents)
	if err != nil {
		return err
	}

	_, err = file.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (s *ArtifactService) GetArtifactPath(artifactory *cosmos.Artifactory, id int) *string {
	path := filepath.Join(artifactory.Path, cosmos.ArtifactNames[id])
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	// For Docker-in-Docker, we have to return the path as it would be on the host.
	path = strings.TrimPrefix(path, cosmos.ArtifactDir)
	path = filepath.Join(os.Getenv("ARTIFACT_DIR"), path)

	return &path
}

func (s *ArtifactService) GetArtifactData(artifactory *cosmos.Artifactory, id int) ([]byte, error) {
	path := filepath.Join(artifactory.Path, cosmos.ArtifactNames[id])

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, cosmos.Errorf(cosmos.ENOTFOUND, "Requested artifact does not exist")
		}
		return nil, err
	}

	return ioutil.ReadFile(path)
}

func (s *ArtifactService) CloseArtifactory(artifactory *cosmos.Artifactory) {
	for _, artifact := range artifactory.Artifacts {
		if artifact != nil {
			artifact.Writer().(*os.File).Close()
		}
	}
}
