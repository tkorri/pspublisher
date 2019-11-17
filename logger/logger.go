package logger

import (
	"fmt"
	"os"
)

type Logger struct {
	verbose bool
	debug   bool
}

func New(verbose bool, debug bool) *Logger {
	return &Logger{verbose: verbose, debug: debug}
}

func (l *Logger) V(format string, v ...interface{}) {
	if l.verbose {
		Println("    "+format, v...)
	}
}

func (l *Logger) D(format string, v ...interface{}) {
	if l.debug {
		Println("  "+format, v...)
	}
}

func (l *Logger) I(format string, v ...interface{}) {
	Println(format, v...)
}

func (l *Logger) E(format string, v ...interface{}) {
	Errorln(format, v...)
}

func Println(format string, v ...interface{}) {
	_, _ = fmt.Fprintf(os.Stdout, format+"\n", v...)
}

func Errorln(format string, v ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", v...)
}
