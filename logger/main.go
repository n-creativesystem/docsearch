package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/n-creativesystem/docsearch/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	TimestampFormat = "2006/01/02 - 15:04:05"
)

var (
	format = strings.ToLower(utils.DefaultGetEnv("LOG_FORMAT_TYPE", "JSON"))
)

func New() *logrus.Logger {
	log := logrus.New()
	SetFormatter(log)
	return log
}

func SetFormatter(log *logrus.Logger) {
	var formatter logrus.Formatter = GetFormatter()
	log.SetFormatter(formatter)
}

func GetFormat() string {
	return format
}

func GetFormatter() logrus.Formatter {
	switch GetFormat() {
	case "text":
		return &logrus.TextFormatter{
			TimestampFormat: TimestampFormat,
		}
	default:
		return &logrus.JSONFormatter{
			TimestampFormat: TimestampFormat,
		}
	}
}

func RestLogger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		fields := logrus.Fields{}
		if logger.GetLevel() == logrus.DebugLevel {
			mpHeader := c.Request.Header.Clone()
			for key, value := range mpHeader {
				if len(value) >= 0 {
					key = strings.ToLower(key)
					k := fmt.Sprintf("req_%s", key)
					v := strings.ToLower(strings.Join(value, ", "))
					fields[k] = v
				}
			}
		}
		c.Next()
		for i, err := range c.Errors {
			logger.Errorf("idx: %d error: %v", i, err)
		}
		param := gin.LogFormatterParams{
			Request: c.Request,
			Keys:    c.Keys,
		}

		param.TimeStamp = time.Now()
		param.Latency = param.TimeStamp.Sub(start)

		param.ClientIP = c.ClientIP()
		param.Method = c.Request.Method
		param.StatusCode = c.Writer.Status()
		param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()

		param.BodySize = c.Writer.Size()

		if raw != "" {
			path = path + "?" + raw
		}

		param.Path = path
		mp := map[string]interface{}{
			"key":      "RBNS",
			"status":   param.StatusCode,
			"latency":  param.Latency,
			"clientIP": param.ClientIP,
			"method":   param.Method,
			"path":     param.Path,
			"Ua":       param.Request.UserAgent(),
		}
		for key, value := range mp {
			fields[key] = value
		}
		if logger.GetLevel() == logrus.DebugLevel {
			mpHeader := c.Writer.Header().Clone()
			for key, value := range mpHeader {
				if len(value) >= 0 {
					key = strings.ToLower(key)
					k := fmt.Sprintf("res_%s", key)
					v := strings.ToLower(strings.Join(value, ", "))
					fields[k] = v
				}
			}
		}
		logger.WithFields(fields).Info("incoming request")
	}
}
