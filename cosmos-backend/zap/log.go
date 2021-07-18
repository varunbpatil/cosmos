package zap

import (
	"cosmos"
	"encoding/json"

	"go.temporal.io/sdk/log"
	"go.uber.org/zap"
)

var (
	_ cosmos.Logger  = (*Logger)(nil)
	_ log.Logger     = (*Logger)(nil)
	_ log.WithLogger = (*Logger)(nil)
)

type Logger struct {
	*zap.SugaredLogger
}

func NewLogger() *Logger {
	rawJSON := []byte(`{
	  "level": "debug",
	  "encoding": "json",
	  "outputPaths": ["stdout", "/tmp/logs"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase",
		"timeKey": "timestamp",
		"timeEncoder": "rfc3339"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic("failed to unmarshal zap config. err: " + err.Error())
	}

	logger, err := cfg.Build()
	if err != nil {
		panic("failed to create a zap logger. err: " + err.Error())
	}

	return &Logger{logger.Sugar()}
}

func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	l.SugaredLogger.Debugw(msg, keyvals...)
}

func (l *Logger) Info(msg string, keyvals ...interface{}) {
	l.SugaredLogger.Infow(msg, keyvals...)
}

func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	l.SugaredLogger.Warnw(msg, keyvals...)
}

func (l *Logger) Error(msg string, keyvals ...interface{}) {
	l.SugaredLogger.Errorw(msg, keyvals...)
}

func (l *Logger) WithKV(keyvals ...interface{}) cosmos.Logger {
	return &Logger{l.SugaredLogger.With(keyvals...)}
}

// This method is there purely so that this logger can be used in Temporal.
// Inside an activity, you can get a cosmos.Logger using activity.GetLogger(ctx).(cosmos.Logger).
// However, inside a workflow, you will not be able to get a cosmos.Logger because workflows use
// a ReplayLogger to be deterministic.
func (l *Logger) With(keyvals ...interface{}) log.Logger {
	return &Logger{l.SugaredLogger.With(keyvals...)}
}
