package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
)

var logger Logger
var once sync.Once

type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
	Panicf(string, ...interface{})
}

func NewLogger() Logger {
	once.Do(func() {

		encoderConfig := zap.NewDevelopmentEncoderConfig()

		config := &zap.Config{
			Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
			Encoding:         "console",
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stdout"},
		}

		if l, err := config.Build(); err != nil {
			panic("Could not create logger")
		} else {
			zap.RedirectStdLog(l)
			logger = l.Sugar()
		}
	})

	return logger
}
