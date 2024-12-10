package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
)

type Logger struct {
	debug  bool
	output *log.Logger
}

var (
	Default *Logger
	once    sync.Once
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
)

func init() {
	once.Do(func() {
		debugMode := os.Getenv("DEBUG") == "true"
		Default = &Logger{
			debug:  debugMode,
			output: log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix),
		}
	})
}

func (l *Logger) Info(v ...interface{}) {
	l.output.SetPrefix(green + "[INFO] " + reset)
	l.output.Println(fmt.Sprint(v...))
}

func (l *Logger) Warning(v ...interface{}) {
	l.output.SetPrefix(yellow + "[WARNING] " + reset)
	l.output.Println(fmt.Sprint(v...))
}

func (l *Logger) Error(v ...interface{}) {
	l.output.SetPrefix(red + "[ERROR] " + reset)
	l.output.Println(fmt.Sprint(v...))
}

func (l *Logger) Debug(v ...interface{}) {
	if l.debug {
		l.output.SetPrefix(blue + "[DEBUG] " + reset)
		l.output.Println(fmt.Sprint(v...))
	}
}

func (l *Logger) SetOutput(outputFile string) error {
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to set logger output: %v", err)
	}
	l.output.SetOutput(file)
	return nil
}
