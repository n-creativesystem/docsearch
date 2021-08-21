package cmd

import (
	"os"
	"time"

	"github.com/n-creativesystem/docsearch/logger"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

type localLogger struct {
	*logrus.Logger
}

func newLogger(format string, level logrus.Level, filename string) logger.LogrusLogger {
	log := logrus.New()
	switch filename {
	case "", os.Stderr.Name():
		log.Out = os.Stderr
	case os.Stdout.Name():
		log.Out = os.Stdout
	default:
		log.Out = &lumberjack.Logger{
			Filename: filename,
		}
	}
	switch format {
	case "text":
		log.Formatter = &logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
		}
	default:
		log.Formatter = &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		}
	}
	return &localLogger{
		Logger: log,
	}
}

func (l *localLogger) Logrus() *logrus.Logger {
	return l.Logger
}

func (l *localLogger) Write(p []byte) (int, error) {
	l.Info(string(p))
	return len(p), nil
}
