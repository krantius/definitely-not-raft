package logging

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

type level int

const (
	TRACE level = iota
	DEBUG
	INFO
	WARNING
	ERROR
)

const (
	format = "2006-01-02 15:04:05"
)

func Trace(msg string) {
	output(TRACE, msg)
}

func Tracef(msg string, args ...interface{}) {
	tmp := make([]interface{}, len(args))

	for i := range args {
		tmp[i] = args[i]
	}

	Trace(fmt.Sprintf(msg, tmp...))
}

func Debug(msg string) {
	output(DEBUG, msg)
}

func Debugf(msg string, args ...interface{}) {
	tmp := make([]interface{}, len(args))

	for i := range args {
		tmp[i] = args[i]
	}

	Debug(fmt.Sprintf(msg, tmp...))
}

func Info(msg string) {
	output(INFO, msg)
}

func Infof(msg string, args ...interface{}) {
	tmp := make([]interface{}, len(args))

	for i := range args {
		tmp[i] = args[i]
	}

	Info(fmt.Sprintf(msg, tmp...))
}

func Warning(msg string) {
	output(WARNING, msg)
}

func Warningf(msg string, args ...interface{}) {
	tmp := make([]interface{}, len(args))

	for i := range args {
		tmp[i] = args[i]
	}

	Warning(fmt.Sprintf(msg, tmp...))
}

func Error(msg string) {
	output(ERROR, msg)
}

func Errorf(msg string, args ...interface{}) {
	tmp := make([]interface{}, len(args))

	for i := range args {
		tmp[i] = args[i]
	}

	Error(fmt.Sprintf(msg, tmp...))
}

func output(l level, msg string) {
	t := time.Now().Format(format)
	switch l {
	case TRACE:
		color.Cyan("%v TRACE %s", t, msg)
	case DEBUG:
		color.Green("%v DEBUG %s", t, msg)
	case INFO:
		color.White("%v INFO %s", t, msg)
	case WARNING:
		color.Blue("%v WARN %s", t, msg)
	case ERROR:
		color.Red("%v ERROR %s", t, msg)
	}
}
