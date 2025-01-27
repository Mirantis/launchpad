package log

import (
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	// Debug turns on debug logging for stdout output.
	Debug = false
	// Trace turns on trace logging for stdout output.
	Trace = false
)

// FormatterWriterHook is a logrus hook implementation that allows customizing both the log stream target and formatter.
type FormatterWriterHook struct {
	Writer    io.Writer
	LogLevels []log.Level
	Formatter log.Formatter
}

// Fire will be called when some logging function is called with current hook
// It will format log entry to string and write it to appropriate writer.
func (hook *FormatterWriterHook) Fire(entry *log.Entry) error {
	line, err := hook.Formatter.Format(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to format log entry: %v", err)
		return fmt.Errorf("unable to format log entry: %w", err)
	}
	if _, err = hook.Writer.Write(line); err != nil {
		return fmt.Errorf("unable to write log entry to writer: %w", err)
	}
	return nil
}

// Levels define on which log levels this hook would trigger.
func (hook *FormatterWriterHook) Levels() []log.Level {
	return hook.LogLevels
}

// NewStdoutHook creates new hook for stdout logging.
func NewStdoutHook() *FormatterWriterHook {
	stdoutHook := &FormatterWriterHook{
		Writer:    os.Stdout,
		Formatter: &log.TextFormatter{DisableTimestamp: false, ForceColors: false, DisableColors: true, DisableQuote: true},
		LogLevels: []log.Level{
			log.InfoLevel,
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
			log.WarnLevel,
		},
	}

	// Add debug level to stdout hook if set by user
	if Trace {
		stdoutHook.LogLevels = append([]log.Level{log.TraceLevel, log.DebugLevel}, stdoutHook.LogLevels...)
	} else if Debug {
		stdoutHook.LogLevels = append([]log.Level{log.DebugLevel}, stdoutHook.LogLevels...)
	}

	return stdoutHook
}

// NewFileHook creates logrus hook for logging all levels to file.
func NewFileHook(logFile *os.File) *FormatterWriterHook {
	fileFormatter := &log.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC822,
	}
	fileHook := &FormatterWriterHook{
		Writer:    logFile,
		Formatter: fileFormatter,
		LogLevels: log.AllLevels,
	}

	return fileHook
}
