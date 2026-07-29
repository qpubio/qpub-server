package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	infrastructure "github.com/qpubio/qpub-server/internal/config/infrastructure"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Service interface {
	Debug(component log.Component, format string, args ...interface{})
	Info(component log.Component, format string, args ...interface{})
	Warn(component log.Component, format string, args ...interface{})
	Error(component log.Component, format string, args ...interface{})
	Fatal(component log.Component, format string, args ...interface{})
}

type service struct {
	config *infrastructure.Logger
	writer *lumberjack.Logger
	mu     sync.Mutex
}

func New(config *infrastructure.Logger) (Service, error) {
	// Ensure absolute path
	logDir := config.Dir
	if !filepath.IsAbs(logDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		logDir = filepath.Join(cwd, logDir)
	}

	// Create log directory with all parent directories
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	writer := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"),
		MaxSize:    config.Rotation.MaxSize,
		MaxBackups: config.Rotation.MaxBackups,
		MaxAge:     config.Rotation.MaxAge,
		Compress:   config.Rotation.Compress,
	}

	return &service{
		config: config,
		writer: writer,
	}, nil
}

func (s *service) log(level log.Level, component log.Component, format string, args ...interface{}) {
	if !s.shouldLog(level) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	timestamp := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)

	if s.config.JSON {
		s.writeJSON(timestamp, level, component, msg)
	} else {
		s.writeText(timestamp, level, component, msg)
	}
}

func (s *service) writeJSON(timestamp string, level log.Level, component log.Component, msg string) {
	entry := map[string]string{
		"timestamp": timestamp,
		"level":     level.String(),
		"component": component.String(),
		"message":   msg,
	}

	if jsonBytes, err := json.Marshal(entry); err == nil {
		s.writer.Write(append(jsonBytes, '\n'))
		os.Stdout.Write(append(jsonBytes, '\n'))
	}
}

func (s *service) writeText(timestamp string, level log.Level, component log.Component, msg string) {
	color := log.GetColorForLevel(level)
	logEntry := fmt.Sprintf("%s %s[%s]\033[0m [%s] %s\n",
		timestamp,
		color,
		level.String(),
		component.String(),
		msg,
	)

	s.writer.Write([]byte(logEntry))
	os.Stdout.Write([]byte(logEntry))
}

func (s *service) shouldLog(level log.Level) bool {
	configLevel := log.ParseLevel(s.config.Level)
	return level >= configLevel
}

func (s *service) Debug(component log.Component, format string, args ...interface{}) {
	s.log(log.Debug, component, format, args...)
}

func (s *service) Info(component log.Component, format string, args ...interface{}) {
	s.log(log.Info, component, format, args...)
}

func (s *service) Warn(component log.Component, format string, args ...interface{}) {
	s.log(log.Warn, component, format, args...)
}

func (s *service) Error(component log.Component, format string, args ...interface{}) {
	s.log(log.Error, component, format, args...)
}

func (s *service) Fatal(component log.Component, format string, args ...interface{}) {
	s.log(log.Fatal, component, format, args...)
	os.Exit(1)
}
