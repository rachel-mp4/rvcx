package log

import (
	"fmt"
	"io"
	"log"
)

type Logger struct {
	debugLogger *log.Logger
	prodLogger *log.Logger
}

func New(w io.Writer, verbose bool) Logger {
	l := Logger{}
	l.prodLogger = log.New(w, "[log]", log.Ldate|log.Ltime)
	if verbose {
		l.debugLogger = log.New(w, "[debug]", log.Ldate|log.Ltime)
	}
	return l
}

func (l *Logger) Deprintln(s string) {
	if l.debugLogger != nil {
		l.debugLogger.Println(s)
	}
}

func (l *Logger) Deprintf(format string, args ...any) {
	l.Deprintln(fmt.Sprintf(format, args...))
}

func (l *Logger) Println(s string) {
	if l.prodLogger != nil {
		l.prodLogger.Println(s)
	}
}

func (l *Logger) Printf(format string, args ...any) {
	l.Println(fmt.Sprintf(format, args...))
}