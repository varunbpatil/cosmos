package cosmos

import (
	"context"
	"fmt"

	"github.com/iancoleman/orderedmap"
)

const (
	MessageTypeRecord            = "RECORD"
	MessageTypeState             = "STATE"
	MessageTypeLog               = "LOG"
	MessageTypeSpec              = "SPEC"
	MessageTypeConnectionStatus  = "CONNECTION_STATUS"
	MessageTypeCatalog           = "CATALOG"
	MessageTypeConfiguredCatalog = "CONFIGURED_CATALOG"

	ConnectionStatusSucceeded = "SUCCEEDED"
	ConnectionStatusFailed    = "FAILED"

	SyncModeFullRefresh = "full_refresh"
	SyncModeIncremental = "incremental"

	DestinationSyncModeAppend      = "append"
	DestinationSyncModeOverwrite   = "overwrite"
	DestinationSyncModeAppendDedup = "append_dedup"
	DestinationSyncModeUpsertDedup = "upsert_dedup"

	LogLevelFatal = "FATAL"
	LogLevelError = "ERROR"
	LogLevelWarn  = "WARN"
	LogLevelInfo  = "INFO"
	LogLevelDebug = "DEBUG"
	LogLevelTrace = "TRACE"
)

type Message struct {
	Type              string             `json:"type,omitempty"`
	Log               *Log               `json:"log,omitempty"`
	Spec              *Spec              `json:"spec,omitempty"`
	ConnectionStatus  *ConnectionStatus  `json:"connectionStatus,omitempty"`
	Catalog           *Catalog           `json:"catalog,omitempty"`
	ConfiguredCatalog *ConfiguredCatalog `json:"configuredCatalog,omitempty"`
	Record            *Record            `json:"record,omitempty"`
	State             *State             `json:"state,omitempty"`
}

type Log struct {
	Level   string `json:"level,omitempty"`
	Message string `json:"message,omitempty"`
}

func (l *Log) String() string {
	// See https://www.lihaoyi.com/post/BuildyourownCommandLinewithANSIescapecodes.html for ansi colors.
	colorPrefix := ""
	colorReset := "\u001b[0m"

	switch l.Level {
	case LogLevelFatal, LogLevelError:
		colorPrefix = "\u001b[31m"
	case LogLevelWarn:
		colorPrefix = "\u001b[33m"
	case LogLevelInfo:
		colorPrefix = "\u001b[32m"
	case LogLevelDebug, LogLevelTrace:
		colorPrefix = "\u001b[34m"
	default:
		panic("Unknown log level")
	}

	return fmt.Sprintf("%s%s%s %s", colorPrefix, l.Level, colorReset, l.Message)
}

type Spec struct {
	ConnectionSpecification       orderedmap.OrderedMap `json:"connectionSpecification,omitempty"`
	DocumentationURL              string                `json:"documentationUrl,omitempty"`
	ChangelogURL                  string                `json:"changelogUrl,omitempty"`
	SupportsIncremental           bool                  `json:"supportsIncremental,omitempty"`
	SupportsNormalization         bool                  `json:"supportsNormalization,omitempty"`
	SupportsDBT                   bool                  `json:"supportsDBT,omitempty"`
	SupportedDestinationSyncModes []string              `json:"supported_destination_sync_modes,omitempty"`
}

type Catalog struct {
	Streams []Stream `json:"streams,omitempty"`
}

type Stream struct {
	Name                    string                `json:"name,omitempty"`
	JSONSchema              orderedmap.OrderedMap `json:"json_schema,omitempty"`
	SupportedSyncModes      []string              `json:"supported_sync_modes,omitempty"`
	SourceDefinedCursor     bool                  `json:"source_defined_cursor,omitempty"`
	DefaultCursorField      []string              `json:"default_cursor_field,omitempty"`
	SourceDefinedPrimaryKey [][]string            `json:"source_defined_primary_key,omitempty"`
	Namespace               *string               `json:"namespace,omitempty"`
}

type ConfiguredCatalog struct {
	Streams []ConfiguredStream `json:"streams,omitempty"`
}

type ConfiguredStream struct {
	Stream              Stream     `json:"stream,omitempty"`
	SyncMode            *string    `json:"sync_mode,omitempty"`
	CursorField         []string   `json:"cursor_field,omitempty"`
	DestinationSyncMode *string    `json:"destination_sync_mode,omitempty"`
	PrimaryKey          [][]string `json:"primary_key,omitempty"`
}

type Record struct {
	Stream    string                 `json:"stream,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	EmittedAt int                    `json:"emitted_at,omitempty"`
	Namespace *string                `json:"namespace,omitempty"`
}

type State struct {
	Data map[string]interface{} `json:"data,omitempty"`
}

type ConnectionStatus struct {
	Status  string  `json:"status,omitempty"`
	Message *string `json:"message,omitempty"`
}

type MessageService interface {
	CreateMessage(ctx context.Context, raw []byte) (*Message, error)
	MessageToForm(ctx context.Context, message *Message, additionalInfo interface{}) *Form
	Validate(ctx context.Context, raw interface{}, message *Message) error
}

func (m *Message) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (s *Stream) IsSyncModeAvailable(syncMode string) bool {
	// SyncModeFullRefresh is supported by all sources even if sync.SupportedSyncModes is empty.
	if syncMode == SyncModeFullRefresh {
		return true
	}
	for _, s := range s.SupportedSyncModes {
		if s == syncMode {
			return true
		}
	}
	return false
}
