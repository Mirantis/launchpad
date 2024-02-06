package pollutil

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	timeStep = 5 * time.Second
)

// abortError signals that polling should be aborted when returned by the run
// function.
type abortError struct {
	Err error
}

func (a abortError) Error() string {
	return fmt.Sprintf("pollutil: aborted: %s", a.Err)
}

func (a abortError) Unwrap() error {
	return a.Err
}

// Abort returns an error which signals that Poll should stop retrying and
// return err.
func Abort(err error) error {
	return abortError{Err: err}
}

// Poll runs the given run function until it returns a nil error, up to retries
// times with a delay of interval in between runs.
//
// The run function can abort polling early by returning Abort(err).
// Usage:
//
//	pollutil.Poll(time.Second, 5, func() error {
//	        err := fallibleOperation()
//	        if errors.Is(err, SomeUnrecoverableError) {
//	                return pollutil.Abort(err)
//	        }
//	        return err
//	})
func Poll(interval time.Duration, retries int, run func() error) (err error) {
	for i := 0; i < retries; i++ {
		err = run()
		if err != nil {
			// Only check if the outermost error is an abortError.
			if abortErr, ok := err.(abortError); ok {
				return abortErr.Unwrap()
			}
			time.Sleep(interval)
		} else {
			return nil
		}
	}

	return fmt.Errorf("polling failed with %d attempts %s apart: %s", retries, interval, err)
}

type PollfConfig struct {
	Interval   time.Duration
	NumRetries int
	LogLevel   logrus.Level
	Format     string
	Args       []interface{}
}

func DefaultPollfConfig(level logrus.Level, format string, args ...interface{}) PollfConfig {
	return PollfConfig{
		Interval:   time.Second,
		NumRetries: 60,
		LogLevel:   level,
		Format:     format,
		Args:       args,
	}
}

func InfoPollfConfig(format string, args ...interface{}) PollfConfig {
	return DefaultPollfConfig(logrus.InfoLevel, format, args...)
}

func DebugPollfConfig(format string, args ...interface{}) PollfConfig {
	return DefaultPollfConfig(logrus.DebugLevel, format, args...)
}

// Pollf configures and returns a polling function. This function accepts a
// second function as input and will poll and log periodically until the
// provided function passes or aborts, or all retries have been exhausted.
// Example usage:
//
//	err = Pollf(PollfConfig{time.Second, 3, logrus.InfoLevel, "stuff %s", []interface{"aaaaaa"}})(func() error {
//	    return testtt()
//	})
func Pollf(pollfConfig PollfConfig) func(func() error) error {
	return func(run func() error) error {
		return Waitf(pollfConfig.LogLevel, pollfConfig.Format, pollfConfig.Args...)(func() error {
			return Poll(pollfConfig.Interval, pollfConfig.NumRetries, run)
		})
	}
}

// Waitf returns a waiting function that is configured with a logging message.
// This function accepts a second function as input and will periodically log
// the message until the provided function exits. The error is passed on from
// the return value of the called function.
// Usage:
//
//	err = Waitf(logrus.InfoLevel, "things: %d", 3)(func() error {
//	    return nil
//	})
func Waitf(level logrus.Level, format string, args ...interface{}) func(func() error) error {
	return func(f func() error) error {
		end := waitf(level, format, args...)
		defer end()
		return f()
	}
}

// waitf starts a separate Go routine which periodically writes waiting info to the
// log. The function returns immediately and provides a "cancel function" to the
// caller. The logging is stopped by calling this cancel function. The cancel function
// is not thread-safe but is safe to call more than once.
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
