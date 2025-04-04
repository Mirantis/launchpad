package cmdbuffer

import (
	"fmt"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
)

var ErrNotALogRusLine = fmt.Errorf("line was not a logrus entry")

// LogEntry interpretation of a logrus record.
type LogEntry struct {
	Level string `json:"level"`
	Time  string `json:"time"`
	Msg   string `json:"msg"`
}

// a regex pattern for matching lines.
var logPattern = regexp.MustCompile(`time="(?P<time>\S*)"\slevel=(?P<level>\S*)\smsg="(?<msg>.*)"`)

// LogrusParseText parse a text logrus entry into its parts.
func LogrusParseText(line string) (LogEntry, error) {
	matches := logPattern.FindStringSubmatch(line)

	logentry := LogEntry{
		Time: time.Now().Format(time.RFC3339),
	}
	var logerror error

	if matches == nil {
		logentry.Level = "info"
		logentry.Msg = line
		logerror = ErrNotALogRusLine
	} else {
		if len(matches) > 3 {
			logentry.Msg = matches[3]
		}
		if len(matches) > 2 {
			logentry.Level = matches[2]
		}
		if len(matches) > 1 {
			logentry.Time = matches[1]
		}
	}

	return logentry, logerror
}

// LogrusLine proxy log a Logrus entry.
func LogrusLine(logentry LogEntry) {
	switch logentry.Level {
	case "debug":
		logrus.Debug(logentry.Msg)
	case "warn":
		logrus.Warn(logentry.Msg)
	case "error":
		logrus.Error(logentry.Msg)
	case "fatal": // you should handle the exception yourself
		logrus.Error(logentry.Msg)
	default: // includes "info"
		logrus.Info(logentry.Msg)
	}
}
