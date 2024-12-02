package logger

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	debug bool
}

var Default *Logger

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
)

func init() {
	Default = &Logger{
		debug: os.Getenv("DEBUG") == "true",
	}
}

func (l *Logger) Info(v ...interface{}) {
	log.Println(green+"[INFO]"+reset, fmt.Sprint(v...))
}

func (l *Logger) Error(v ...interface{}) {
	log.Println(red+"[ERROR]"+reset, fmt.Sprint(v...))
}

func (l *Logger) Debug(v ...interface{}) {
	if l.debug {
		log.Println(blue+"[DEBUG]"+reset, fmt.Sprint(v...))
	}
}
