package logger

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type LogLevel int

const (
	DEBUG  LogLevel = 0
	INFO   LogLevel = 1
	WARN   LogLevel = 2
	ERROR  LogLevel = 3
	lOGSYS LogLevel = 100
)

var logLevelMap = map[LogLevel]string{
	DEBUG:  "DEBUG",
	INFO:   "INFO",
	WARN:   "WARN",
	ERROR:  "ERROR",
	lOGSYS: "LOGSYS",
}
var logColerMap = map[LogLevel]int{
	DEBUG:  34,
	INFO:   32,
	WARN:   33,
	ERROR:  31,
	lOGSYS: 31,
}

func colorLevel(level LogLevel) string {
	return fmt.Sprint("\033[1;", logColerMap[level], "m", logLevelMap[level], "\033[0m")
}

type message struct {
	Level   LogLevel
	Message string
	File    string
	Line    int
	Time    time.Time
}

func newMessage(level LogLevel, msg string) *message {
	_, file, line, _ := runtime.Caller(2)
	return &message{level, msg, file, line, time.Now()}
}

type Writter interface {
	Write(msg []byte) (int, error)
}

type Logger struct {
	level   LogLevel
	msgs    chan *message
	size    int
	closed  bool
	exit    chan bool
	time    time.Time
	writter Writter
}

// NewLogger create a new logger
func NewLogger(level LogLevel, size int, w Writter) *Logger {
	l := &Logger{
		level:   level,
		msgs:    make(chan *message, size),
		closed:  false,
		size:    size,
		time:    time.Now(),
		exit:    make(chan bool),
		writter: w,
	}
	if l.writter == nil {
		l.writter = os.Stdout
	}
	return l.run()
}

// NewLogger create a new logger with writter
func NewLoggerWithWritter(level LogLevel, size int, w Writter) *Logger {
	return &Logger{
		level:   level,
		msgs:    make(chan *message, size),
		closed:  false,
		size:    size,
		time:    time.Now(),
		writter: w,
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}
func (l *Logger) GetLevel() LogLevel {
	return l.level
}
func (l *Logger) SetWritter(w Writter) {
	l.writter = w
}

// write message to channel
func (l *Logger) write(msg *message) {
	if l.closed {
		return
	} else if msg.Level >= l.level {
		l.msgs <- msg
	}
}

func (l *Logger) Log(level LogLevel, msg string) {
	l.write(newMessage(level, msg))
}
func (l *Logger) DEBUG(msg string) {
	l.write(newMessage(DEBUG, msg))
}

func (l *Logger) INFO(msg string) {
	l.write(newMessage(INFO, msg))
}

func (l *Logger) WARN(msg string) {
	l.write(newMessage(WARN, msg))
}

func (l *Logger) ERROR(msg string) {
	l.write(newMessage(ERROR, msg))
}

// read message from channel
func (l *Logger) read() *message {
	return <-l.msgs
}

// Write message to writter
func (l *Logger) run() *Logger {
	go l.catchSignal()
	l.Log(lOGSYS, "Logger started")
	go func() {
		for !l.closed || len(l.msgs) > 0 {
			msg := l.read()
			l.writter.Write([]byte(fmt.Sprintf("%s\t%s\t%s:%d\t%s\n", colorLevel(msg.Level), msg.Time.Format("2006-01-02 15:04:05"), msg.File, msg.Line, msg.Message)))
		}
		l.exit <- true
	}()
	return l
}
func (l *Logger) catchSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1,
		syscall.SIGUSR2, syscall.SIGTSTP)
	<-sig
	l.Close()
	os.Exit(0)
}

// Close logger
func (l *Logger) Close() {
	l.Log(lOGSYS, "Logger closed")
	l.closed = true
	close(l.msgs)
	<-l.exit
}
