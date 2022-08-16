package sqs

import (
	"fmt"
	"strings"

	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/reactor"
)

const (
	defaultMaxNumberOfMessages = 10 // Default limit of messages can be read from SQS
	waitTimeSeconds            = 15 // Seconds to keep open the connection to SQS
)

// SQSPlugin struct for SQS Input plugin
type SQSPlugin struct {
	r       *reactor.Reactor
	l       *sqsListen
	URL     string
	Region  string
	Profile string
	done    chan struct{}
}

// NewOrGet create a new SQS plugin and relate it with the Reactor
func NewOrGet(r *reactor.Reactor, c map[string]interface{}) (*SQSPlugin, error) {

	p := &SQSPlugin{
		r:    r,
		done: make(chan struct{}),
	}

	for k, v := range c {
		switch strings.ToLower(k) {
		case "url":
			p.URL = v.(string)
		case "region":
			p.Region = v.(string)
		case "profile":
			p.Profile = v.(string)
		}
	}

	if p.URL == "" {
		return nil, fmt.Errorf("SQS ERROR: URL not found or invalid")
	}

	if p.Region == "" {
		return nil, fmt.Errorf("SQS ERROR: Region not found or invalid")
	}

	if nl, ok := connPool.Load(p.URL); !ok {
		var err error
		p.l, err = newSQSListen(r, c)
		if err != nil {
			return nil, err
		}
		connPool.Store(p.URL, p.l)
	} else {
		p.l = nl.(*sqsListen)
	}

	p.l.AddOrUpdate(r)

	return p, nil
}

// Put is not needed in SQS
func (p *SQSPlugin) Put(v lib.Msg) error {
	return nil
}

// Exit stops the pooling from SQS
func (p *SQSPlugin) Exit() {
	p.l.Exit()
	connPool.Delete(p.URL)
}

// Stops listening
func (p *SQSPlugin) Stop() {
	p.l.Stop()
}

func (p *SQSPlugin) Done(v lib.Msg, status bool) {
	p.l.Done(v, status)
}
