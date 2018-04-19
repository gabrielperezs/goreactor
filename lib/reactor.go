package lib

import (
	"log"
	"strings"
	"sync/atomic"
	"time"
)

var (
	counters uint64
)

type Reactor struct {
	I               Input
	O               Output
	Ch              chan *Msg
	id              uint64
	Concurrent      int
	ConcurrentSleep time.Duration
}

func NewReactor(icfg interface{}) *Reactor {
	r := &Reactor{
		Ch:              make(chan *Msg, 100),
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
				r.ConcurrentSleep = 5 * time.Second
			}
		}
	}

	if r.Concurrent <= 0 {
		r.Concurrent = 1
	}

	log.Printf("D: %#v", r)

	return r
}

func (r *Reactor) GetId() uint64 {
	return r.id
}

func (r *Reactor) Run() {
	for i := 0; i < r.Concurrent; i++ {
		go r.listener()
	}
}

func (r *Reactor) listener() {
	for msg := range r.Ch {

		//log.Printf("R [%d]: GET: %s", r.id, string(msg.B))

		if r.O == nil {
			continue
		}

		if err := r.O.Run(msg); err != nil {
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

		// log.Printf("R [%d]: DONE (sleep %s)", r.id, r.ConcurrentSleep)

		if r.ConcurrentSleep.Seconds() > 0 {
			time.Sleep(r.ConcurrentSleep)
		}
	}
}
