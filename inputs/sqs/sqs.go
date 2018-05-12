package sqs

import (
	"fmt"
	"strings"

	"github.com/gabrielperezs/goreactor/lib"
)

const (
	maxNumberOfMessages = 10 // Limit of messages can be read from SQS
	waitTimeSeconds     = 15 // Seconds to keep open the connection to SQS
)

type SQSPlugin struct {
	r       *lib.Reactor
	l       *sqsListen
	URL     string
	Region  string
	Profile string
	exiting uint32
	done    chan struct{}
}

func NewOrGet(r *lib.Reactor, c map[string]interface{}) (*SQSPlugin, error) {

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

func (p *SQSPlugin) Delete(msg *lib.Msg) error {
	return p.l.Delete(msg)
}

// Put is not needed in SQS
func (p *SQSPlugin) Put(msg *lib.Msg) error {
	return nil
}

func (p *SQSPlugin) Exit() {
	p.l.Exit()
	connPool.Delete(p.URL)
	return
}
