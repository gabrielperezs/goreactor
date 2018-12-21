package lib

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var (
	logMu       = &sync.Mutex{}
	logFileName string
	logFile     *os.File
	logCh       = make(chan []byte, 1024)
	logRotateCh = make(chan bool)
	logDoneCh   = make(chan bool)
)

// LogReload will run a goroutine to handle the logs, that are the
// StdOut and StdErr from the output plugins
func LogReload(name string) {
	if logFile == nil {
		go LogListener(name)
		return
	}

	logFileName = name
	logRotateCh <- true
}

// LogListener will store in files the output of the output plugins
func LogListener(name string) {
	defer func() {
		close(logRotateCh)
		close(logDoneCh)
	}()

	logFileName = name
	f, err := OpenLogFile(logFileName)
	if err != nil {
		log.Panicln(err)
	}
	for {
		select {
		case b := <-logCh:
			if _, err := f.Write(b); err != nil {
				log.Printf("ERROR write log file: %s", err)
				break
			}
		case <-logRotateCh:
			f.Close()
			var err error
			f, err = OpenLogFile(logFileName)
			if err != nil {
				log.Printf("ERROR log: %s", err)
			}
		case <-logDoneCh:
			close(logCh)
			f.Close()
			return
		}
	}
}

// OpenLogFile will create or reuse a file to store the logs
func OpenLogFile(name string) (*os.File, error) {
	var err error
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("ERROR opening log file: %s", err)
		return nil, err
	}
	return f, nil
}

// LogClose will finish the log system
func LogClose() {
	logDoneCh <- true
}

// LogRotate will rotate the underline files
func LogRotate() {
	logRotateCh <- true
}

// LogWrite create a slice of bytes that will be write in the files
func LogWrite(b []byte) {
	l := []byte(time.Now().UTC().String())
	l = append(l, []byte(" ")...)
	l = append(l, b...)
	l = append(l, []byte("\n")...)
	logCh <- l
}

// NewReactorLog create a log method for the reactors
func NewReactorLog(rid uint64, tid uint64) ReactorLog {
	r := ReactorLog{
		rid:    rid,
		tid:    tid,
		prefix: []byte(fmt.Sprintf("RID %d RTID %08d OUTPUT | ", rid, tid)),
	}

	r.rep = append([]byte("\n"), r.prefix...)

	return r
}

// ReactorLog is the struct that will be associate to an specific reactor
type ReactorLog struct {
	prefix []byte
	rep    []byte
	rid    uint64
	tid    uint64
}

// Write will be called by the reactor and this bytes will be sent to the general log channel
func (rl ReactorLog) Write(b []byte) (int, error) {
	for _, l := range bytes.Split(b, []byte("\n")) {
		if len(l) == 0 {
			continue
		}

		n := []byte(time.Now().UTC().String())
		n = append(n, []byte(" ")...)
		n = append(n, rl.prefix...)

		if bytes.HasSuffix(l, []byte("\n")) {
			logCh <- append(n, l...)
		} else {
			logCh <- append(append(n, l...), []byte("\n")...)
		}
	}
	return len(b), nil
}

// WriteStrings is the same as prev function but to receive strings
func (rl ReactorLog) WriteStrings(s string) (int, error) {
	rl.Write([]byte(s))
	return len(s), nil
}
