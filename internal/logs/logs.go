package logs

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Options struct {
	Level          string
	Output         string // stdout|file|both
	FilePath       string
	FileMaxSizeMB  int
	FileMaxBackups int
	FileMaxAgeDays int
	FileCompress   bool
}

func New(level string) zerolog.Logger { // backward-compat
	return NewWithOptions(Options{Level: level, Output: "stdout"})
}

func NewWithOptions(opt Options) zerolog.Logger {
	lvl := zerolog.InfoLevel
	switch strings.ToLower(opt.Level) {
	case "debug":
		lvl = zerolog.DebugLevel
	case "info":
		lvl = zerolog.InfoLevel
	case "warn":
		lvl = zerolog.WarnLevel
	case "error":
		lvl = zerolog.ErrorLevel
	}
	zerolog.TimeFieldFormat = time.RFC3339

	var writers []io.Writer
	if opt.Output == "stdout" || opt.Output == "both" || opt.Output == "" {
		writers = append(writers, os.Stdout)
	}
	if opt.Output == "file" || opt.Output == "both" {
		if opt.FilePath == "" {
			opt.FilePath = "./logs/ads-analyzer.log"
		}
		_ = os.MkdirAll(dirname(opt.FilePath), 0o755)
		lw := &lumberjack.Logger{
			Filename:   opt.FilePath,
			MaxSize:    max(1, opt.FileMaxSizeMB),
			MaxBackups: max(0, opt.FileMaxBackups),
			MaxAge:     max(0, opt.FileMaxAgeDays),
			Compress:   opt.FileCompress,
		}
		writers = append(writers, lw)
	}
	var w io.Writer
	if len(writers) == 0 {
		w = os.Stdout
	} else if len(writers) == 1 {
		w = writers[0]
	} else {
		w = zerolog.MultiLevelWriter(writers...)
	}
	logger := zerolog.New(w).With().Timestamp().Logger().Level(lvl)
	return logger
}

func dirname(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return "."
}
