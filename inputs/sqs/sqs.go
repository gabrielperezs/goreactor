package sqs

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gabrielperezs/goreactor/lib"
)

const (
	maxNumberOfMessages = 10 // Limit of messages can be read from SQS
	waitTimeSeconds     = 15 // Seconds to keep open the connection to SQS
)

var connPool sync.Map

type SQSPlugin struct {
	r       *lib.Reactor
	URL     string
	Region  string
	Profile string

	sess *session.Session
	svc  *sqs.SQS

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

	log.Printf("%+v", p)

	if p.URL == "" {
		return nil, fmt.Errorf("SQS ERROR: URL not found or invalid")
	}

	if p.Region == "" {
		return nil, fmt.Errorf("SQS ERROR: Region not found or invalid")
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: p.Profile,
	})
	if err != nil {
		return nil, err
	}

	p.svc = sqs.New(sess, &aws.Config{Region: aws.String(p.Region)})

	go p.listen()

	return p, nil
}

func (p *SQSPlugin) listen() {
	defer func() {
		p.done <- struct{}{}
	}()

	for {
		if atomic.LoadUint32(&p.exiting) > 0 {
			return
		}

		params := &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(p.URL),
			MaxNumberOfMessages: aws.Int64(maxNumberOfMessages),
			WaitTimeSeconds:     aws.Int64(waitTimeSeconds),
		}

		resp, err := p.svc.ReceiveMessage(params)

		if err != nil {
			log.Printf("ERROR: AWS session on %s - %s", p.URL, err)
			time.Sleep(15 * time.Second)
			continue
		}

		for _, msg := range resp.Messages {
			b := []byte(*msg.Body)
			jsonParsed, err := gabs.ParseJSON(b)
			if err == nil && jsonParsed.Exists("Message") {
				p.r.Ch <- strings.Replace(jsonParsed.S("Message").String(), "\\\"", "\"", -1)
				continue
			}

			p.r.Ch <- b
		}
	}
}

func (p *SQSPlugin) Exit() error {
	atomic.AddUint32(&p.exiting, 1)
	<-p.done
	return nil
}
