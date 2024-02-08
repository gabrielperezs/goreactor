package jsonreactorlog

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gabrielperezs/goreactor/lib"
)

var (
	newLine            = []byte("\n")
	jsonReactorLogPool = &sync.Pool{
		New: func() interface{} {
			return &JSONReactorLog{
				w: strings.Builder{},
			}
		},
	}
)

// NewJSONReactorLog returns a logger that sends json lines to the specified LogStream
func NewJSONReactorLog(logStream lib.LogStream, hostname string, rid uint64, tid uint64) *JSONReactorLog {
	r := jsonReactorLogPool.Get().(*JSONReactorLog)
	r.Host = hostname
	r.logStream = logStream
	r.RID = rid
	r.TID = tid
	r.Status = ""
	r.st = time.Now()
	r.Timestamp = r.st.Unix()
	return r
}

type ReactorLog interface {
	Write(b []byte) (int, error)
	Start(pid int, s string)
	SetLabel(string)
	SetHash(string)
}

// JSONReactorLog lets you log as json lines.
type JSONReactorLog struct {
	Host      string  `json:",omitempty"`
	Label     string  `json:",omitempty"`
	Hash      string  `json:",omitempty"`
	Pid       int     `json:",omitempty"`
	RID       uint64  `json:",omitempty"`
	TID       uint64  `json:",omitempty"`
	Line      uint64  // Do not omit line number on line 0
	Output    string  `json:",omitempty"`
	Status    string  `json:",omitempty"`
	Error     string  `json:",omitempty"`
	Elapse    float64 `json:",omitempty"`
	Timestamp int64   `json:",omitempty"`
	st        time.Time
	w         strings.Builder
	logStream lib.LogStream

	initialized bool

	buff bytes.Buffer

	sync.Mutex
}

func (rl *JSONReactorLog) SetLabel(value string) {
	rl.Label = value
}

func (rl *JSONReactorLog) SetHash(value string) {
	rl.Hash = value
}

// Write will be called by the reactor and this bytes will be sent to the general log channel
func (rl *JSONReactorLog) Write(b []byte) (int, error) {
	rl.Lock()
	defer rl.Unlock()
	if !rl.initialized {
		return rl.buff.Write(b)
	}
	if rl.buff.Len() > 0 {
		rl.handleWriteBytes(rl.buff.Bytes())
		rl.buff.Reset()
	}

	rl.handleWriteBytes(b)
	return len(b), nil
}

func (rl *JSONReactorLog) handleWriteBytes(b []byte) {
	for _, l := range b {
		if l == newLine[0] {
			rl.printJSON()
		} else {
			rl.w.WriteByte(l)
		}
	}
}

// Start change status and write the initial command
func (rl *JSONReactorLog) Start(pid int, s string) {
	rl.Lock()
	defer rl.Unlock()

	rl.Pid = pid
	rl.initialized = true
	rl.Status = "CMD"
	rl.w.WriteString(s)
	rl.printJSON()
	rl.Status = "RUN"
}

// Done write in the logs the elapse time for the current execution
func (rl *JSONReactorLog) Done(err error) {
	rl.Lock()
	defer rl.Unlock()
	if !rl.initialized {
		rl.handleWriteBytes(rl.buff.Bytes()) // It was never Started
	}

	rl.Status = "END"
	if err != nil {
		rl.Error = err.Error()
	}
	// https://github.com/golang/go/issues/5491#issuecomment-66079585
	rl.Elapse = time.Now().Sub(rl.st).Seconds()
	rl.printJSON()
	rl.reset()
}

func (rl *JSONReactorLog) printJSON() {
	rl.Output = rl.w.String()
	rl.w.Reset()

	b, err := json.Marshal(rl)
	if err != nil {
		log.Printf("INTERNAL ERROR: %s", err.Error())
		return
	}
	ls := rl.logStream
	if ls != nil {
		ls.Send(b)
	}
	rl.Output = ""
	rl.Line++
}

func (rl *JSONReactorLog) reset() {
	rl.Host = ""
	rl.Label = ""
	rl.Hash = ""
	rl.Pid = 0
	rl.RID = 0
	rl.TID = 0
	rl.Line = 0
	rl.Output = ""
	rl.Status = ""
	rl.Error = ""
	rl.Elapse = 0
	rl.Timestamp = 0
	rl.logStream = nil
	rl.w.Reset()
	rl.buff.Reset()
	jsonReactorLogPool.Put(rl)
}
