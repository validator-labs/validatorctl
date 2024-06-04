package logging

import (
	"fmt"
	"io"
	logging "log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
)

// The global logger's standard methods (i.e., log.Infof, log.Debugf, etc.)
// write log entries to disk.
//
//   file location: ~/.validator/logs/validator.log
//
// The log.InfoCLI method logs entries to the console. It is used to guide users
// through an interactive TUI experience.

var (
	log                 *logrus.Logger
	cliLog              = pterm.DefaultLogger
	logFile, statusFile string
	Newline             = true
)

func init() {
	log = &logrus.Logger{
		Out: io.Discard,
		Formatter: &logrus.TextFormatter{
			FullTimestamp: true,
		},
	}
}

func SetLevel(logLevel string) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logging.Fatalf("error setting log level: %v", err)
	}
	log.SetLevel(level)
}

func GetLevel() logrus.Level {
	return log.Level
}

func SetOutput(runLoc string) {
	logFile = filepath.Join(runLoc, "logs", "validator.log")
	statusFile = filepath.Join(runLoc, "status", "status")

	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600) //#nosec
	if err != nil {
		logging.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
}

// logContext recovers the original caller context of each log message
func logContext() *logrus.Entry {
	if pc, file, line, ok := runtime.Caller(2); ok {
		file = file[strings.LastIndex(file, "/")+1:]
		funcFull := runtime.FuncForPC(pc).Name()
		funcName := funcFull[strings.LastIndex(funcFull, ".")+1:]
		entry := log.WithField("src", fmt.Sprintf("%s:%s:%d", file, funcName, line))
		return entry
	}
	return nil
}

// Debug ...
func Debug(format string, v ...interface{}) {
	entry := logContext()
	entry.Debugf(format, v...)
}

// Info ...
func Info(format string, v ...interface{}) {
	entry := logContext()
	entry.Infof(format, v...)
}

// Warn ...
func Warn(format string, v ...interface{}) {
	entry := logContext()
	entry.Warnf(format, v...)
}

// Error ...
func Error(format string, v ...interface{}) {
	entry := logContext()
	entry.Errorf(format, v...)
}

// FatalCLI prints a message to the terminal & exits
func FatalCLI(msg string, args ...any) {
	entry := logContext()
	ptermLog(cliLog.Fatal, entry, msg, args...)
	entry.Fatal(msg, args)
}

// InfoCLI prints an info message to the terminal & creates a log entry
func InfoCLI(format string, v ...interface{}) {
	printToConsole(format, v...)

	entry := logContext()
	entry.Infof(format, v...)
}

// ErrorCLI prints an error message to the terminal & creates a log entry
func ErrorCLI(msg string, args ...any) {
	entry := logContext()
	ptermLog(cliLog.Error, entry, msg, args...)
	entry.Info(msg, args)
}

func ptermLog(f func(string, ...[]pterm.LoggerArgument), entry *logrus.Entry, msg string, args ...any) {
	// pterm.Logger does not support an odd number of arguments
	numArgs := len(args)
	if numArgs%2 == 0 {
		f(msg, cliLog.Args(args...))
	} else {
		errMsg := "error: invalid (odd) number of arguments (%d) provided to ErrorCLI"
		printToConsole(errMsg, numArgs)
		entry.Fatalf(errMsg, numArgs)
	}
}

func printToConsole(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	fmt.Fprint(os.Stdout, strings.TrimSuffix(s, "\n"))
	if Newline {
		fmt.Fprintf(os.Stdout, "\n")
	}
}

func Header(s string) {
	HeaderCustom(s, pterm.BgCyan, pterm.FgBlack)
}

func HeaderCustom(s string, bgColor, textColor pterm.Color) {
	fmt.Fprintf(os.Stdout, "\n")
	pterm.DefaultHeader.
		WithMargin(15).
		WithBackgroundStyle(pterm.NewStyle(bgColor)).
		WithTextStyle(pterm.NewStyle(textColor)).
		WithFullWidth(true).
		Println(s)
	fmt.Fprintf(os.Stdout, "\n")
}

// Out returns the io.Writer used to write messages to the console
func Out() *os.File {
	return os.Stdout
}
