package adapter

import (
	"bytes"
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/sirupsen/logrus"
)

type HcLogAdapter struct {
	log  logger.LogrusLogger
	name string
	args []interface{}
}

var _ hclog.Logger = (*HcLogAdapter)(nil)

func NewHcLogAdapter(log logger.LogrusLogger, name string, args ...interface{}) *HcLogAdapter {
	return &HcLogAdapter{
		log:  log,
		name: name,
		args: args,
	}
}

func (a *HcLogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Trace:
		a.Trace(msg, args...)
	case hclog.Debug:
		a.Debug(msg, args...)
	case hclog.Info:
		a.Info(msg, args...)
	case hclog.Warn:
		a.Warn(msg, args...)
	case hclog.Error:
		a.Error(msg, args...)
	}
}

func (a *HcLogAdapter) Trace(msg string, args ...interface{}) {
	a.createEntry(args).Trace(msg)
}

func (a *HcLogAdapter) Debug(msg string, args ...interface{}) {
	a.createEntry(args).Debug(msg)
}

func (a *HcLogAdapter) Info(msg string, args ...interface{}) {
	a.createEntry(args).Info(msg)
}

func (a *HcLogAdapter) Warn(msg string, args ...interface{}) {
	a.createEntry(args).Warn(msg)
}

func (a *HcLogAdapter) Error(msg string, args ...interface{}) {
	a.createEntry(args).Error(msg)
}

func (a *HcLogAdapter) IsTrace() bool {
	return a.shouldEmit(logrus.TraceLevel)
}

func (a *HcLogAdapter) IsDebug() bool {
	return a.shouldEmit(logrus.DebugLevel)
}

func (a *HcLogAdapter) IsInfo() bool {
	return a.shouldEmit(logrus.InfoLevel)
}

func (a *HcLogAdapter) IsWarn() bool {
	return a.shouldEmit(logrus.WarnLevel)
}

func (a *HcLogAdapter) IsError() bool {
	return a.shouldEmit(logrus.ErrorLevel)
}

func (a *HcLogAdapter) ImpliedArgs() []interface{} {
	return a.args
}

func (a *HcLogAdapter) With(args ...interface{}) hclog.Logger {
	return &HcLogAdapter{
		log:  a.log,
		args: concatFields(a.args, args),
	}
}

func concatFields(a, b []interface{}) []interface{} {
	c := make([]interface{}, len(a)+len(b))
	copy(c, a)
	copy(c[len(a):], b)
	return c
}

func (a *HcLogAdapter) Name() string {
	return a.name
}

func (a *HcLogAdapter) Named(name string) hclog.Logger {
	var newName bytes.Buffer
	if a.name != "" {
		newName.WriteString(a.name)
		newName.WriteString(".")
	}
	newName.WriteString(name)
	return a.ResetNamed(newName.String())
}

func (a *HcLogAdapter) ResetNamed(name string) hclog.Logger {
	return &HcLogAdapter{
		log:  a.log,
		name: name,
	}
}

func (a *HcLogAdapter) SetLevel(level hclog.Level) {
	switch level {
	case hclog.Trace:
		a.log.Logrus().SetLevel(logrus.TraceLevel)
	case hclog.Debug:
		a.log.Logrus().SetLevel(logrus.DebugLevel)
	case hclog.Info:
		a.log.Logrus().SetLevel(logrus.InfoLevel)
	case hclog.Warn:
		a.log.Logrus().SetLevel(logrus.WarnLevel)
	case hclog.Error:
		a.log.Logrus().SetLevel(logrus.ErrorLevel)
	}
}

func (a *HcLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.New(a.log.Logrus().Out, "docsearch", log.LstdFlags)
}

func (a *HcLogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return a.log.Logrus().Out
}

func (a *HcLogAdapter) shouldEmit(level logrus.Level) bool {
	return a.log.Logrus().Level >= level
}

func (a *HcLogAdapter) createEntry(args []interface{}) *logrus.Entry {
	if len(args)%2 != 0 {
		args = append(args, "<unknown>")
	}

	fields := make(logrus.Fields)
	for i := 0; i < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			continue
		}
		v := args[i+1]
		fields[k] = v
	}

	return a.log.Logrus().WithFields(fields)
}
