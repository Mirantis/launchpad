package log

import (
	log "github.com/sirupsen/logrus"
)

// LogHost is an interface that can be implemented to provide a host name to the logger.
type Host interface {
	Name() string
}

type HostLogger struct {
	Host Host
}

func (l *HostLogger) withHost() *log.Entry {
	if l.Host == nil {
		return &log.Entry{}
	}
	return log.WithField("host", l.Host.Name())
}

// Trace logs a message at level Trace on the standard logger.
func (l *HostLogger) Trace(args ...interface{}) {
	l.withHost().Trace(args...)
}

// Debug logs a message at level Debug on the standard logger.
func (l *HostLogger) Debug(args ...interface{}) {
	l.withHost().Debug(args...)
}

// Print logs a message at level Info on the standard logger.
func (l *HostLogger) Print(args ...interface{}) {
	l.withHost().Print(args...)
}

// Info logs a message at level Info on the standard logger.
func (l *HostLogger) Info(args ...interface{}) {
	l.withHost().Info(args...)
}

// Warn logs a message at level Warn on the standard logger.
func (l *HostLogger) Warn(args ...interface{}) {
	l.withHost().Warn(args...)
}

// Warning logs a message at level Warn on the standard logger.
func (l *HostLogger) Warning(args ...interface{}) {
	l.withHost().Warning(args...)
}

// Error logs a message at level Error on the standard logger.
func (l *HostLogger) Error(args ...interface{}) {
	l.withHost().Error(args...)
}

// Panic logs a message at level Panic on the standard logger.
func (l *HostLogger) Panic(args ...interface{}) {
	l.withHost().Panic(args...)
}

// Fatal logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func (l *HostLogger) Fatal(args ...interface{}) {
	l.withHost().Fatal(args...)
}

// Tracef logs a message at level Trace on the standard logger.
func (l *HostLogger) Tracef(format string, args ...interface{}) {
	l.withHost().Tracef(format, args...)
}

// Debugf logs a message at level Debug on the standard logger.
func (l *HostLogger) Debugf(format string, args ...interface{}) {
	l.withHost().Debugf(format, args...)
}

// Printf logs a message at level Info on the standard logger.
func (l *HostLogger) Printf(format string, args ...interface{}) {
	l.withHost().Printf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func (l *HostLogger) Infof(format string, args ...interface{}) {
	l.withHost().Infof(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func (l *HostLogger) Warnf(format string, args ...interface{}) {
	l.withHost().Warnf(format, args...)
}

// Warningf logs a message at level Warn on the standard logger.
func (l *HostLogger) Warningf(format string, args ...interface{}) {
	l.withHost().Warningf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func (l *HostLogger) Errorf(format string, args ...interface{}) {
	l.withHost().Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func (l *HostLogger) Panicf(format string, args ...interface{}) {
	l.withHost().Panicf(format, args...)
}

// Fatalf logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func (l *HostLogger) Fatalf(format string, args ...interface{}) {
	l.withHost().Fatalf(format, args...)
}

// Traceln logs a message at level Trace on the standard logger.
func (l *HostLogger) Traceln(args ...interface{}) {
	l.withHost().Traceln(args...)
}

// Debugln logs a message at level Debug on the standard logger.
func (l *HostLogger) Debugln(args ...interface{}) {
	l.withHost().Debugln(args...)
}

// Println logs a message at level Info on the standard logger.
func (l *HostLogger) Println(args ...interface{}) {
	l.withHost().Println(args...)
}

// Infoln logs a message at level Info on the standard logger.
func (l *HostLogger) Infoln(args ...interface{}) {
	l.withHost().Infoln(args...)
}

// Warnln logs a message at level Warn on the standard logger.
func (l *HostLogger) Warnln(args ...interface{}) {
	l.withHost().Warnln(args...)
}

// Warningln logs a message at level Warn on the standard logger.
func (l *HostLogger) Warningln(args ...interface{}) {
	l.withHost().Warningln(args...)
}

// Errorln logs a message at level Error on the standard logger.
func (l *HostLogger) Errorln(args ...interface{}) {
	l.withHost().Errorln(args...)
}

// Panicln logs a message at level Panic on the standard logger.
func (l *HostLogger) Panicln(args ...interface{}) {
	l.withHost().Panicln(args...)
}

// Fatalln logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func (l *HostLogger) Fatalln(args ...interface{}) {
	l.withHost().Fatalln(args...)
}
