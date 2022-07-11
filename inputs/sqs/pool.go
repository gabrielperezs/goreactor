package sqs

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/reactor"
)

var MessageSystemAttributeNameSentTimestamp = sqs.MessageSystemAttributeNameSentTimestamp

var connPool sync.Map

type sqsListen struct {
	sync.Mutex
	URL                 string
	Region              string
	Profile             string
	MaxNumberOfMessages int64

	svc *sqs.SQS

	exiting uint32
	done    chan struct{}

	broadcastCh sync.Map
	pendings    map[string]int
}

func newSQSListen(r *reactor.Reactor, c map[string]interface{}) (*sqsListen, error) {

	p := &sqsListen{
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
		case "maxnumberofmessages":
			p.MaxNumberOfMessages, _ = v.(int64)
		}
	}

	if p.MaxNumberOfMessages == 0 {
		p.MaxNumberOfMessages = defaultMaxNumberOfMessages
	}

	if p.URL == "" {
		return nil, fmt.Errorf("SQS ERROR: URL not found or invalid")
	}

	if p.Region == "" {
		return nil, fmt.Errorf("SQS ERROR: Region not found or invalid")
	}

	log.Printf("SQS NEW %s", p.URL)

	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:           p.Profile,
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}
	p.broadcastCh.Store(r, true)
	p.pendings = make(map[string]int)

	p.svc = sqs.New(sess, &aws.Config{Region: aws.String(p.Region)})

	go p.listen()

	return p, nil
}

func (p *sqsListen) AddOrUpdate(r *reactor.Reactor) {
	p.broadcastCh.Store(r, true)
}

func (p *sqsListen) listen() {
	defer func() {
		p.done <- struct{}{}
	}()

	for {
		if atomic.LoadUint32(&p.exiting) > 0 {
			tries := 0
			for len(p.pendings) > 0 {
				time.Sleep(time.Second)
				tries++
				if tries > 120 { // Wait no more than 120 seconds, the usual max
					break
				}
			}
			return
		}

		params := &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(p.URL),
			MaxNumberOfMessages: aws.Int64(p.MaxNumberOfMessages),
			WaitTimeSeconds:     aws.Int64(waitTimeSeconds),
			AttributeNames:      []*string{&MessageSystemAttributeNameSentTimestamp},
		}

		resp, err := p.svc.ReceiveMessage(params)

		if err != nil {
			log.Printf("ERROR: AWS session on %s - %s", p.URL, err)
			time.Sleep(15 * time.Second)
			continue
		}

		for _, msg := range resp.Messages {
			// Flag to delete the message if don't match with at least one reactor condition
			atLeastOneValid := false

			timestamp, ok := msg.Attributes[sqs.MessageSystemAttributeNameSentTimestamp]
			var sentTimestamp int64
			if ok && timestamp != nil {
				sentTimestamp, _ = strconv.ParseInt(*timestamp, 10, 64)
			}

			m := &Msg{
				SQS: p.svc,
				B:   []byte(*msg.Body),
				M: &sqs.DeleteMessageBatchRequestEntry{
					Id:            msg.MessageId,
					ReceiptHandle: msg.ReceiptHandle,
				},
				URL:           aws.String(p.URL),
				SentTimestamp: sentTimestamp,
			}

			jsonParsed, err := gabs.ParseJSON(m.B)
			if err == nil && jsonParsed.Exists("Message") {
				s := strings.Replace(jsonParsed.S("Message").String(), "\\\"", "\"", -1)
				s = strings.TrimPrefix(s, "\"")
				s = strings.TrimSuffix(s, "\"")
				m.B = []byte(s)
			}

			p.broadcastCh.Range(func(k, v interface{}) bool {
				if err := k.(*reactor.Reactor).MatchConditions(m); err != nil {
					return true
				}
				atLeastOneValid = true
				k.(*reactor.Reactor).Ch <- m
				p.AddPending(m)
				return true
			})

			// We delete this message if is invalid for all the reactors
			if !atLeastOneValid {
				log.Printf("Invalid message from %s, deleted: %s", p.URL, *msg.Body)
				p.Delete(m)
			}
		}
	}
}

func (p *sqsListen) Stop() {
	if atomic.LoadUint32(&p.exiting) > 0 {
		return
	}
	atomic.AddUint32(&p.exiting, 1)
	log.Printf("SQS Input Stopping %s", p.URL)
}

func (p *sqsListen) Exit() error {
	p.Stop()
	<-p.done
	log.Printf("SQS EXIT %s", p.URL)
	return nil
}

func (p *sqsListen) Delete(v lib.Msg) (err error) {
	msg, ok := v.(*Msg)
	if !ok {
		log.Printf("ERROR SQS Delete: invalid message %+v", v)
		return
	}

	if msg.SQS == nil || msg == nil {
		return
	}
	if _, err = msg.SQS.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      msg.URL,
		ReceiptHandle: msg.M.ReceiptHandle,
	}); err != nil {
		log.Printf("ERROR: %s - %s", *msg.URL, err)
	}
	p.DelPending(v)
	return
}

func (p *sqsListen) AddPending(m lib.Msg) {
	p.Lock()
	defer p.Unlock()
	msg, ok := m.(*Msg)
	if !ok {
		return
	}
	id := *msg.M.ReceiptHandle
	v, ok := p.pendings[id]
	if ok {
		v += 1
	} else {
		v = 1
	}
	p.pendings[id] = v
}

func (p *sqsListen) DelPending(m lib.Msg) {
	p.Lock()
	defer p.Unlock()
	msg, ok := m.(*Msg)
	if !ok {
		return
	}
	id := *msg.M.ReceiptHandle

	v, ok := p.pendings[id]
	if !ok {
		return
	}
	if v <= 1 {
		delete(p.pendings, id)
		return
	}
	v -= 1
	p.pendings[id] = v
}
