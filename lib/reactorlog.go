package lib

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	logMu       = &sync.Mutex{}
	logFileName string
	logFile     *os.File
	logCh       = make(chan []byte, 1024)
	logRotateCh = make(chan bool)
	logDoneCh   = make(chan bool)
)

func LogReload(name string) {
	if logFile == nil {
		go LogListener(name)
		return
	}

	logFileName = name
	logRotateCh <- true
}

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

func OpenLogFile(name string) (*os.File, error) {
	var err error
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("ERROR opening log file: %s", err)
		return nil, err
	}
	return f, nil
}

func LogClose() {
	logDoneCh <- true
}

func LogRotate() {
	logRotateCh <- true
}

func NewReactorLog(rid uint64, tid uint64) ReactorLog {
	r := ReactorLog{
		rid:    rid,
		tid:    tid,
		prefix: []byte(fmt.Sprintf("RID %d RTID %08d OUTPUT | ", rid, tid)),
	}

	r.rep = append([]byte("\n"), r.prefix...)

	return r
}

type ReactorLog struct {
	prefix []byte
	rep    []byte
	rid    uint64
	tid    uint64
}

func (rl ReactorLog) Write(b []byte) (int, error) {
	for _, l := range bytes.Split(b, []byte("\n")) {
		if len(l) == 0 {
			continue
		}

		if bytes.HasSuffix(l, []byte("\n")) {
			logCh <- append(rl.prefix, l...)
		} else {
			logCh <- append(append(rl.prefix, l...), []byte("\n")...)
		}
	}
	return len(b), nil
}

func (rl ReactorLog) WriteStrings(s string) (int, error) {
	rl.Write([]byte(s))
	return len(s), nil
}
