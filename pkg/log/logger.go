package log

import (
	"fmt"
	"log"
	"sync"
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
	_Red     string = "%s\033[1;31m%s\033[0m"
	_Green   string = "%s\033[1;32m%s\033[0m"
	_Yellow  string = "%s\033[1;33m%s\033[0m"
	_Blue    string = "%s\033[1;34m%s\033[0m"
	_Magenta string = "%s\033[1;35m%s\033[0m"
	_Cyan    string = "%s\033[1;36m%s\033[0m"
	_White   string = "%s\033[1;37m%s\033[0m"
)

var logger Logger = Logger{prefix: "", level: LevelInfo, lock: &sync.Mutex{}}

// Logger .
type Logger struct {
	level  uint8
	prefix string
	lock   *sync.Mutex
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
	logger.lock.Lock()
	if logger.level >= LevelDebug {
		log.SetPrefix(fmt.Sprintf(_Cyan, logger.prefix, " | DEBUG | "))
		log.Printf(format, v...)
	}
	logger.lock.Unlock()
}

// Info .
func Info(format string, v ...interface{}) {
	logger.lock.Lock()
	if logger.level >= LevelInfo {
		log.SetPrefix(fmt.Sprintf(_Green, logger.prefix, " |  INFO | "))
		log.Printf(format, v...)
	}
	logger.lock.Unlock()
}

// Warn .
func Warn(format string, v ...interface{}) {
	logger.lock.Lock()
	if logger.level >= LevelWarn {
		log.SetPrefix(fmt.Sprintf(_Yellow, logger.prefix, " |  Warn | "))
		log.Printf(format, v...)
	}
	logger.lock.Unlock()
}

// Error .
func Error(format string, v ...interface{}) {
	logger.lock.Lock()
	if logger.level >= LevelError {
		log.SetPrefix(fmt.Sprintf(_Red, logger.prefix, " | ERROR | "))
		log.Printf(format, v...)
	}
	logger.lock.Unlock()
}

// Fatal is equivalent to log.Printf() followed by a call to os.Exit(1).
func Fatal(format string, v ...interface{}) {
	logger.lock.Lock()
	if logger.level >= LevelFatal {
		log.SetPrefix(fmt.Sprintf(_Magenta, logger.prefix, " | FATAL | "))
		logger.lock.Unlock()
		log.Fatalf(format, v...)
	}
}
