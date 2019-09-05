package lib

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	counters uint64

	// ErrInvalidMsgForPlugin error
	ErrInvalidMsgForPlugin = fmt.Errorf("This message is not valid for this output")
)

// Reactor is the struct where we keep the relation betwean Input plugins
// and the Output plugins. Also contains the configuration for concurrency...
type Reactor struct {
	mu           sync.Mutex
	I            Input
	O            Output
	Ch           chan Msg
	id           uint64
	tid          uint64
	Concurrent   int
	Delay        time.Duration
	Label        string
	Hostname     string
	listeners    int64
	nextDeadline time.Time
	done         chan bool
	logStream    LogStream
}

// NewReactor will create a reactor with the configuration
func NewReactor(icfg interface{}) *Reactor {
	r := &Reactor{
		id:         atomic.AddUint64(&counters, 1),
		Concurrent: 0,
		Delay:      0,
		done:       make(chan bool),
	}

	r.Reload(icfg)

	r.Ch = make(chan Msg, r.Concurrent)

	log.Printf("Reactor %d concurrent %d, delay %s", r.id, r.Concurrent, r.Delay)

	return r
}

// Reload will replace the old configuration with new parameters
func (r *Reactor) Reload(icfg interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, ok := icfg.(map[string]interface{})
	if !ok {
		log.Printf("ERROR Reactor config")
		return
	}

	for k, v := range cfg {
		switch strings.ToLower(k) {
		case "concurrent":
			r.Concurrent = int(v.(int64))
		case "label":
			r.Label = v.(string)
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
}

// SetLogStreams define what streams use for the log
func (r *Reactor) SetLogStreams(lg LogStream) error {
	r.logStream = lg
	return nil
}

// SetHostname define the hostname
func (r *Reactor) SetHostname(name string) error {
	r.Hostname = name
	return nil
}

// MatchConditions will call to the MatchConditions of the Output
func (r *Reactor) MatchConditions(msg Msg) error {
	return r.O.MatchConditions(msg)
}

// GetID to obtain the ID of the reactor
func (r *Reactor) GetID() uint64 {
	return r.id
}

// Start will run the current reactor in a go routine based on the
// concurrency configuration
func (r *Reactor) Start() {
	for i := 0; i < r.Concurrent; i++ {
		go r.listener()
	}
}

// Exit will close the interaction betwean the Input plugin and the Output
// plugin, and finishing the reactor
func (r *Reactor) Exit() {
	r.I.Exit()
	close(r.Ch)
	r.O.Exit()
	if r.logStream != nil {
		r.logStream.Exit()
	}
}

func (r *Reactor) listener() {
	defer func() {
		r.done <- true
		//log.Printf("Done listener reactor")
	}()

	for msg := range r.Ch {
		if r.O != nil {
			r.run(msg)
		}
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

func (r *Reactor) run(msg Msg) {
	r.deadline()

	var err error
	rl := NewReactorLog(r.logStream, r.Hostname, r.id, atomic.AddUint64(&r.tid, 1))
	err = r.O.Run(rl, msg)
	defer rl.Done(err)

	if err == nil {
		if err := r.I.Delete(msg); err != nil {
			log.Printf("INTERNAL ERROR: %s", err)
		}
		return
	}

	if err == ErrInvalidMsgForPlugin {
		return
	}

	if err := r.I.Put(msg); err != nil {
		log.Printf("INTERNAL ERROR: %s", err)
	}
}
