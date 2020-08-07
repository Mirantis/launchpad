package pollutil

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	timeStep = 5 * time.Second
)

// Poll runs the given function until it passes
func Poll(interval time.Duration, retries int, run func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	for i := 0; i < retries; i++ {
		err = run()
		if err != nil {
			time.Sleep(interval)
		} else {
			return nil
		}
	}

	return fmt.Errorf("polling failed with %d attempts %s apart: %s", retries, interval, err)
}

// PollfConfig defines a poll config
type PollfConfig struct {
	Interval   time.Duration
	NumRetries int
	LogLevel   logrus.Level
	Format     string
	Args       []interface{}
}

// DefaultPollfConfig returns a default PollfConfig
func DefaultPollfConfig(level logrus.Level, format string, args ...interface{}) PollfConfig {
	return PollfConfig{
		Interval:   time.Second,
		NumRetries: 60,
		LogLevel:   level,
		Format:     format,
		Args:       args,
	}
}

// InfoPollfConfig returns a DefaultPollfConfig for logging at info log level
func InfoPollfConfig(format string, args ...interface{}) PollfConfig {
	return DefaultPollfConfig(logrus.InfoLevel, format, args...)
}

// DebugPollfConfig returns a DefaultPollfConfig for logging at debug log level
func DebugPollfConfig(format string, args ...interface{}) PollfConfig {
	return DefaultPollfConfig(logrus.DebugLevel, format, args...)
}

// Pollf polls and prints periodically until the given function passes
// Example usage:
// err = Pollf(time.Second, 3, "stuff %s", "aaaaaa")(func() error {
//     return testtt()
// })
func Pollf(pollfConfig PollfConfig) func(func() error) error {
	return func(run func() error) error {
		return Waitf(pollfConfig.LogLevel, pollfConfig.Format, pollfConfig.Args...)(func() error {
			return Poll(pollfConfig.Interval, pollfConfig.NumRetries, run)
		})
	}
}

// Waitf takes a function and a format string and periodically prints that
// string until the function exits. Panics are handled correctly. The error is
// passed on from the return value of the function.
// Usage:
// err = Waitf("things: %d", 3)(func() error {
//     return nil
// })
func Waitf(level logrus.Level, format string, args ...interface{}) func(func() error) error {
	return func(f func() error) error {
		end := waitf(level, format, args...)
		defer end()
		return f()
	}
}

// waitf prints meaningful waiting info to the log until you call the func it
// returns. The returned func is not thread-safe but it's safe to call more than
// once.
func waitf(level logrus.Level, format string, args ...interface{}) func() {
	elapsed := time.Duration(0)
	ticker := time.NewTicker(timeStep)
	finishWait := make(chan struct{})
	done := false

	logAtLevel(level, format, args...)
	go func() {
		for {
			select {
			case <-finishWait:
				return
			case <-ticker.C:
				elapsed += timeStep
				logAtLevel(level, fmt.Sprintf("%s; elapsed: %s", format, elapsed), args...)
			}
		}
	}()

	return func() {
		if !done {
			close(finishWait)
			done = true
		}
	}
}

func logAtLevel(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.DebugLevel:
		logrus.Debugf(format, args...)
	case logrus.InfoLevel:
		logrus.Infof(format, args...)
	case logrus.WarnLevel:
		logrus.Warnf(format, args...)
	case logrus.ErrorLevel:
		logrus.Errorf(format, args...)
	case logrus.FatalLevel:
		logrus.Errorf(format, args...)
	default:
		panic(fmt.Sprintf("doesn't support this log level: %s", level))
	}
}
