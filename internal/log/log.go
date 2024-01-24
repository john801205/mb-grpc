package log

import (
	"fmt"
	"log"
	"os"
)

var (
	debugLogger = log.New(os.Stdout, "debug ", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)
	infoLogger  = log.New(os.Stdout, "info ", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)
	warnLogger  = log.New(os.Stdout, "warn ", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)
	errorLogger = log.New(os.Stdout, "error ", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)
)

func Debug(v ...any) {
	debugLogger.Output(2, fmt.Sprintln(v...))
}

func Debugf(format string, v ...any) {
	debugLogger.Output(2, fmt.Sprintf(format, v...))
}

func Info(v ...any) {
	infoLogger.Output(2, fmt.Sprintln(v...))
}

func Infof(format string, v ...any) {
	infoLogger.Output(2, fmt.Sprintf(format, v...))
}

func Warn(v ...any) {
	warnLogger.Output(2, fmt.Sprintln(v...))
}

func Warnf(format string, v ...any) {
	warnLogger.Output(2, fmt.Sprintf(format, v...))
}

func Error(v ...any) {
	errorLogger.Output(2, fmt.Sprintln(v...))
}

func Errorf(format string, v ...any) {
	errorLogger.Output(2, fmt.Sprintf(format, v...))
}

func Fatal(v ...any) {
	errorLogger.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func Fatalf(format string, v ...any) {
	errorLogger.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
