package log

import (
	"fmt"
	"log"
)

// log level
const (
	LevelFatal = iota + 1
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
)

const (
	Red     string = "%s\033[1;31m%s\033[0m"
	Green   string = "%s\033[1;32m%s\033[0m"
	Yellow  string = "%s\033[1;33m%s\033[0m"
	Blue    string = "%s\033[1;34m%s\033[0m"
	Magenta string = "%s\033[1;35m%s\033[0m"
	Cyan    string = "%s\033[1;36m%s\033[0m"
	White   string = "%s\033[1;37m%s\033[0m"
)

var logger Logger = Logger{prefix: "[GoSM]", level: LevelInfo}

// Logger .
type Logger struct {
	prefix string
	level  uint8
}

// SetLevel .
func SetLevel(level uint8) error {
	if level < LevelFatal || level > LevelDebug {
		return fmt.Errorf("Logger: log level invalid")
	}
	logger.level = level
	return nil
}

// SetPrefix .
func SetPrefix(prefix string) {
	logger.prefix = prefix
}

// Debug .
func Debug(format string, v ...interface{}) {
	if logger.level >= LevelDebug {
		log.SetPrefix(fmt.Sprintf(Cyan, logger.prefix, " | DEBUG | "))
		log.Printf(format, v...)
	}
}

// Info .
func Info(format string, v ...interface{}) {
	if logger.level >= LevelInfo {
		log.SetPrefix(fmt.Sprintf(Green, logger.prefix, " |  INFO | "))
		log.Printf(format, v...)
	}
}

// Warn .
func Warn(format string, v ...interface{}) {
	if logger.level >= LevelWarn {
		log.SetPrefix(fmt.Sprintf(Yellow, logger.prefix, " |  Warn | "))
		log.Printf(format, v...)
	}
}

// Error .
func Error(format string, v ...interface{}) {
	if logger.level >= LevelError {
		log.SetPrefix(fmt.Sprintf(Red, logger.prefix, " | ERROR | "))
		log.Printf(format, v...)
	}
}

// Fatal is equivalent to log.Printf() followed by a call to os.Exit(1).
func Fatal(format string, v ...interface{}) {
	if logger.level >= LevelFatal {
		log.SetPrefix(fmt.Sprintf(Magenta, logger.prefix, " | FATAL | "))
		log.Fatalf(format, v...)
	}
}
