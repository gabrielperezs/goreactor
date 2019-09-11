package lib

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	newLine        = []byte("\n")
	reactorLogPool = &sync.Pool{
		New: func() interface{} {
			return &ReactorLog{
				w: strings.Builder{},
			}
		},
	}
)

// NewReactorLog create a log method for the reactors
func NewReactorLog(logStream LogStream, hostname string, rid uint64, tid uint64) *ReactorLog {
	r := reactorLogPool.Get().(*ReactorLog)
	r.Host = hostname
	r.logStream = logStream
	r.RID = rid
	r.TID = tid
	r.Status = ""
	r.st = time.Now()
	r.Timestamp = r.st.Unix()
	return r
}

// ReactorLog is the struct that will be associate to an specific reactor
type ReactorLog struct {
	Host      string  `json:",omitempty"`
	Label     string  `json:",omitempty"`
	RID       uint64  `json:",omitempty"`
	TID       uint64  `json:",omitempty"`
	Output    string  `json:",omitempty"`
	Status    string  `json:",omitempty"`
	Error     string  `json:",omitempty"`
	Elapse    float64 `json:",omitempty"`
	Timestamp int64   `json:",omitempty"`
	st        time.Time
	w         strings.Builder
	logStream LogStream
}

// Write will be called by the reactor and this bytes will be sent to the general log channel
func (rl *ReactorLog) Write(b []byte) (int, error) {
	for _, l := range b {
		if l == newLine[0] {
			rl.printJSON()
		} else {
			rl.w.WriteByte(l)
		}
	}
	return len(b), nil
}

// Start change status and write the initial command
func (rl *ReactorLog) Start(s string) {
	rl.Status = "CMD"
	rl.w.WriteString(s)
	rl.printJSON()
	rl.Status = "RUN"
}

// WriteStrings is the same as prev function but to receive strings
func (rl *ReactorLog) WriteStrings(s string) (int, error) {
	rl.w.WriteString(s)
	rl.printJSON()
	return len(s), nil
}

// Done write in the logs the elapse time for the current execution
func (rl *ReactorLog) Done(err error) {
	rl.Status = "END"
	if err != nil {
		rl.Error = err.Error()
	}
	// https://github.com/golang/go/issues/5491#issuecomment-66079585
	rl.Elapse = time.Now().Sub(rl.st).Seconds()
	rl.printJSON()
	rl.reset()
}

func (rl *ReactorLog) printJSON() {
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
}

func (rl *ReactorLog) reset() {
	rl.Host = ""
	rl.Label = ""
	rl.RID = 0
	rl.TID = 0
	rl.Output = ""
	rl.Status = ""
	rl.Error = ""
	rl.Elapse = 0
	rl.Timestamp = 0
	rl.logStream = nil
	rl.w.Reset()
	reactorLogPool.Put(rl)
}
