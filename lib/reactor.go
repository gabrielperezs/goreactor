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
	mu           sync.Mutex
	I            Input
	O            Output
	Ch           chan *Msg
	id           uint64
	tid          uint64
	Concurrent   int
	Delay        time.Duration
	running      int64
	nextDeadline time.Time
}

func NewReactor(icfg interface{}) *Reactor {
	r := &Reactor{
		id:         atomic.AddUint64(&counters, 1),
		Concurrent: 0,
		Delay:      0,
	}

	cfg, ok := icfg.(map[string]interface{})
	if !ok {
		log.Panicf("ERROR NewReactor config")
	}

	for k, v := range cfg {
		switch strings.ToLower(k) {
		case "concurrent":
			r.Concurrent = int(v.(int64))
		case "delay":
			var err error
			r.Delay, err = time.ParseDuration(v.(string))
			if err != nil {
				r.Delay = 0
			}
		}
	}

	if r.Concurrent <= 0 {
		r.Concurrent = 1
	}

	r.Ch = make(chan *Msg, r.Concurrent)

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

func (r *Reactor) Exit() {
	r.I.Exit()
	r.O.Exit()
	close(r.Ch)
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
	if r.nextDeadline.Add(r.Delay).Before(n) {
		r.nextDeadline = n
	} else {
		r.nextDeadline = r.nextDeadline.Add(r.Delay)
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
