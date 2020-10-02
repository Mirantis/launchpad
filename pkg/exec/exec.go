package exec

import (
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
)

// Option is a functional option for the exec package
type Option func(*Options)

// Options is a collection of exec options
type Options struct {
	Stdin        string
	LogInfo      bool
	LogDebug     bool
	LogError     bool
	LogCommand   bool
	StreamOutput bool
	RedactFunc   func(string) string
	Output       *string
}

// LogCmd is for logging the command to be executed
func (o *Options) LogCmd(prefix, cmd string) {
	var msg string
	if o.LogCommand {
		msg = fmt.Sprintf("%s: executing `%s`", prefix, cmd)
	} else {
		msg = fmt.Sprintf("%s: executing [REDACTED]", prefix)
	}
	log.Debugf(msg)
}

// LogStdin is for logging information about command stdin input
func (o *Options) LogStdin(prefix string) {
	if o.Stdin == "" || !o.LogDebug {
		return
	}

	if len(o.Stdin) > 256 {
		o.LogDebugf("%s: writing %d bytes to command stdin", prefix, len(o.Stdin))
	} else {
		o.LogDebugf("%s: writing %d bytes to command stdin: %s", prefix, len(o.Stdin), o.Redact(o.Stdin))
	}
}

// LogDebugf is a conditional debug logger
func (o *Options) LogDebugf(s string, args ...interface{}) {
	if o.LogDebug {
		log.Debugf(s, args...)
	}
}

// LogInfof is a conditional info logger
func (o *Options) LogInfof(s string, args ...interface{}) {
	if o.LogInfo {
		log.Infof(s, args...)
	}
}

// LogErrorf is a conditional error logger
func (o *Options) LogErrorf(s string, args ...interface{}) {
	if o.LogError {
		log.Errorf(s, args...)
	}
}

// AddOutput is for appending / displaying output of the command
func (o *Options) AddOutput(prefix, s string) {
	if o.Output != nil {
		*o.Output += s
	}

	if o.StreamOutput {
		log.Infof("%s: %s", prefix, o.Redact(s))
	} else {
		o.LogDebugf("%s: %s", prefix, o.Redact(s))
	}
}

// Redact is for filtering text based on the exec option "redact"
func (o *Options) Redact(s string) string {
	if o.RedactFunc == nil {
		return s
	}
	return o.RedactFunc(s)
}

// Stdin exec option for sending data to the command through stdin
func Stdin(t string) Option {
	return func(o *Options) {
		o.Stdin = t
	}
}

// Output exec option for setting output string target
func Output(output *string) Option {
	return func(o *Options) {
		o.Output = output
	}
}

// StreamOutput exec option for sending the command output to info log
func StreamOutput() Option {
	return func(o *Options) {
		o.StreamOutput = true
	}
}

// HideCommand exec option for hiding the command-string and stdin contents from the logs
func HideCommand() Option {
	return func(o *Options) {
		o.LogCommand = false
	}
}

// HideOutput exec option for hiding the command output from logs
func HideOutput() Option {
	return func(o *Options) {
		o.LogDebug = false
	}
}

// Sensitive exec option for disabling all logging of the command
func Sensitive() Option {
	return func(o *Options) {
		o.LogDebug = false
		o.LogInfo = false
		o.LogError = false
		o.LogCommand = false
	}
}

// Redact exec option for defining a redact regexp pattern that will be replaced with [REDACTED] in the logs
func Redact(rexp string) Option {
	return func(o *Options) {
		re := regexp.MustCompile(rexp)
		o.RedactFunc = func(s2 string) string {
			return re.ReplaceAllString(s2, "[REDACTED]")
		}
	}
}

// Build returns an instance of Options
func Build(opts ...Option) *Options {
	options := &Options{
		Stdin:        "",
		LogInfo:      false,
		LogCommand:   true,
		LogDebug:     true,
		LogError:     true,
		StreamOutput: false,
		Output:       nil,
	}

	for _, o := range opts {
		o(options)
	}

	return options
}
