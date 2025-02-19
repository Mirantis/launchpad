package cmdbuffer

import (
	"regexp"

	"github.com/sirupsen/logrus"
)

// LogEntry intrepretation of a logrus record
type LogEntry struct {
	Level string `json:"level"` 
	Time  string `json:"time"`
	Msg   string `json:"msg"`
}


// a regex pattern for matching lines
var logPattern = regexp.MustCompile(`time="(?P<time>\S*)"\slevel=(?P<level>\S*)\smsg="(?<msg>.*)"`)
// LogrusParseText parse a text logrus entry into its parts
func LogrusParseText(line string) LogEntry {
	m := logPattern.FindStringSubmatch(line)

	return LogEntry{
		Time: m[1],
		Level: m[2],
		Msg: m[3],
	}
}

// LogrusLine proxy log a Logrus entry
func LogrusLine(le LogEntry) {
	switch (le.Level) {
	case "debug":
		logrus.Debug(le.Msg)
	case "warn":
		logrus.Warn(le.Msg)
	case "error":
		logrus.Error(le.Msg)
	case "info":
		logrus.Info(le.Msg)
	}
}
