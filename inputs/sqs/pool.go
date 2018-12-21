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

var connPool sync.Map

type sqsListen struct {
	URL     string
	Region  string
	Profile string

	sess *session.Session
	svc  *sqs.SQS

	exiting uint32
	done    chan struct{}

	broadcastCh sync.Map
}

func newSQSListen(r *lib.Reactor, c map[string]interface{}) (*sqsListen, error) {

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
		}
	}

	if p.URL == "" {
		return nil, fmt.Errorf("SQS ERROR: URL not found or invalid")
	}

	if p.Region == "" {
		return nil, fmt.Errorf("SQS ERROR: Region not found or invalid")
	}

	log.Printf("SQS NEW %s", p.URL)

	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: p.Profile,
	})
	if err != nil {
		return nil, err
	}
	p.broadcastCh.Store(r, true)

	p.svc = sqs.New(sess, &aws.Config{Region: aws.String(p.Region)})

	go p.listen()

	return p, nil
}

func (p *sqsListen) AddOrUpdate(r *lib.Reactor) {
	p.broadcastCh.Store(r, true)
}

func (p *sqsListen) listen() {
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
			// Flag to delete the message if don't match with at least one reactor condition
			atLeastOneValid := false

			m := &Msg{
				SQS: p.svc,
				B:   []byte(*msg.Body),
				M: &sqs.DeleteMessageBatchRequestEntry{
					Id:            msg.MessageId,
					ReceiptHandle: msg.ReceiptHandle,
				},
				URL: aws.String(p.URL),
			}

			jsonParsed, err := gabs.ParseJSON(m.B)
			if err == nil && jsonParsed.Exists("Message") {
				s := strings.Replace(jsonParsed.S("Message").String(), "\\\"", "\"", -1)
				s = strings.TrimPrefix(s, "\"")
				s = strings.TrimSuffix(s, "\"")
				m.B = []byte(s)
			}

			p.broadcastCh.Range(func(k, v interface{}) bool {
				if err := k.(*lib.Reactor).MatchConditions(m); err != nil {
					return true
				}
				atLeastOneValid = true
				k.(*lib.Reactor).Ch <- m
				return true
			})

			// We delete this message if is invalid for all the reactors
			if !atLeastOneValid {
				lib.LogWrite([]byte(fmt.Sprintf("Invalid message, deleted: %s", *msg.Body)))
				p.Delete(m)
			}
		}
	}
}

func (p *sqsListen) Exit() error {
	if atomic.LoadUint32(&p.exiting) > 0 {
		return nil
	}
	atomic.AddUint32(&p.exiting, 1)
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
	return
}
