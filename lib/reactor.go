package lib

import (
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	counters uint64
)

type Reactor struct {
	mu              sync.Mutex
	I               Input
	O               Output
	Ch              chan *Msg
	id              uint64
	tid             uint64
	Concurrent      int
	ConcurrentSleep time.Duration
	running         int64
	nextDeadline    time.Time
}

func NewReactor(icfg interface{}) *Reactor {
	r := &Reactor{
		id:              atomic.AddUint64(&counters, 1),
		Concurrent:      0,
		ConcurrentSleep: 0,
	}

	cfg, ok := icfg.(map[string]interface{})
	if !ok {
		log.Panicf("ERROR NewReactor config")
	}

	for k, v := range cfg {
		switch strings.ToLower(k) {
		case "concurrent":
			r.Concurrent = int(v.(int64))
		case "concurrentsleep":
			var err error
			r.ConcurrentSleep, err = time.ParseDuration(v.(string))
			if err != nil {
				r.ConcurrentSleep = 0
			}
		}
	}

	if r.Concurrent <= 0 {
		r.Concurrent = 1
	}

	r.Ch = make(chan *Msg, r.Concurrent)

	log.Printf("D: %#v", r)

	return r
}

func (r *Reactor) GetId() uint64 {
	return r.id
}

func (r *Reactor) Start() {
	for i := 0; i < r.Concurrent; i++ {
		go r.listener()
	}
}

func (r *Reactor) Write(b []byte) (int, error) {
	log.Printf("R [%d]: OUTPUT: %s", r.id, string(b))
	return len(b), nil
}

func (r *Reactor) listener() {
	for msg := range r.Ch {
		if r.O == nil {
			continue
		}

		r.run(msg)
	}
}

func (r *Reactor) deadline() {
	r.mu.Lock()
	n := time.Now()
	if r.nextDeadline.Add(r.ConcurrentSleep).Before(n) {
		r.nextDeadline = n
	} else {
		r.nextDeadline = r.nextDeadline.Add(r.ConcurrentSleep)
	}
	sleep := r.nextDeadline.Sub(n)
	r.mu.Unlock()

	if sleep.Seconds() <= 1 {
		return
	}

	time.Sleep(sleep)
}

func (r *Reactor) run(msg *Msg) {
	r.deadline()

	atomic.AddInt64(&r.running, 1)
	defer func() {
		atomic.AddInt64(&r.running, -1)
	}()

	rl := NewReactorLog(r.id, atomic.AddUint64(&r.tid, 1))
	if err := r.O.Run(rl, msg); err != nil {
		switch err {
		case InvalidMsgForPlugin:
		default:
			if err := r.I.Put(msg); err != nil {
				log.Printf("R [%d]: ERROR PUT: %s", r.id, string(msg.B))
			}
		}
	} else {
		if err := r.I.Delete(msg); err != nil {
			log.Printf("R [%d]: ERROR DELETE: %s", r.id, string(msg.B))
		}
	}
}

func NewReactorLog(rid uint64, tid uint64) ReactorLog {
	return ReactorLog{
		rid: rid,
		tid: tid,
	}
}

type ReactorLog struct {
	rid uint64
	tid uint64
}

func (rl ReactorLog) Write(b []byte) (int, error) {
	log.Printf("RID %d RTID %08d OUTPUT | %s", rl.rid, rl.tid, string(b))
	return len(b), nil
}

func (rl ReactorLog) WriteStrings(s string) (int, error) {
	log.Printf("RID %d RTID %08d OUTPUT | %s", rl.rid, rl.tid, s)
	return len(s), nil
}
