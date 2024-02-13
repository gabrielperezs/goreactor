package sqs

import (
	"context"
	"fmt"
	"log"
	"math"
	"runtime"
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
	"github.com/gallir/dynsemaphore"
)

var MessageSystemAttributeNameSentTimestamp = sqs.MessageSystemAttributeNameSentTimestamp

var connPool sync.Map

type sqsListen struct {
	sync.Mutex
	url                 string
	region              string
	profile             string
	maxNumberOfMessages int64
	noBlocking          bool // If true, the input will not block waiting for a reactor to finish

	svc *sqs.SQS

	exiting  uint32
	exited   bool
	exitedMu sync.Mutex
	done     chan bool

	broadcastCh       sync.Map
	pendings          map[string]int
	messError         map[string]bool
	maxQueuedMessages *dynsemaphore.DynSemaphore // Max of goroutines wating to send the message
}

func newSQSListen(r *reactor.Reactor, c map[string]interface{}) (*sqsListen, error) {

	p := &sqsListen{
		done: make(chan bool),
	}

	for k, v := range c {
		switch strings.ToLower(k) {
		case "url":
			p.url = v.(string)
		case "region":
			p.region = v.(string)
		case "profile":
			p.profile = v.(string)
		case "maxnumberofmessages":
			p.maxNumberOfMessages, _ = v.(int64)
		case "noblocking":
			p.noBlocking, _ = v.(bool)
		}
	}

	if p.maxNumberOfMessages == 0 {
		p.maxNumberOfMessages = defaultMaxNumberOfMessages
	}

	if p.url == "" {
		return nil, fmt.Errorf("SQS ERROR: URL not found or invalid")
	}

	if p.region == "" {
		return nil, fmt.Errorf("SQS ERROR: Region not found or invalid")
	}

	log.Printf("SQS NEW %s", p.url)

	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:           p.profile,
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}
	p.broadcastCh.Store(r, r.Concurrent)
	p.pendings = make(map[string]int)
	p.messError = make(map[string]bool)

	p.svc = sqs.New(sess, &aws.Config{Region: aws.String(p.region)})
	p.maxQueuedMessages = dynsemaphore.New(0)
	p.updateConcurrency()

	go p.listen()

	return p, nil
}

func (p *sqsListen) AddOrUpdate(r *reactor.Reactor) {
	p.broadcastCh.Store(r, r.Concurrent)
	p.updateConcurrency()
}

func (p *sqsListen) updateConcurrency() {
	total := 0
	p.broadcastCh.Range(func(k, v interface{}) bool {
		total += v.(int)
		return true
	})
	maxPendings := total
	if maxPendings < runtime.NumCPU() {
		maxPendings = runtime.NumCPU()
	}
	maxPendings = maxPendings * 2 // Its means x*nreactors max pending messages via goroutines, What's the right number?
	log.Printf("SQS: total concurrency: %d, max pending in-flight messages: %d", total, maxPendings)
	p.maxQueuedMessages.SetConcurrency(total)
}

func (p *sqsListen) listen() {
	defer func() {
		p.done <- true
		log.Printf("SQS EXIT %s", p.url)
	}()

	for {
		if atomic.LoadUint32(&p.exiting) > 0 {
			log.Printf("SQS Listener Stopped %s", p.url)
			tries := 0
			for len(p.pendings) > 0 {
				time.Sleep(time.Second)
				tries++
				if tries > 120 { // Wait no more than 120 seconds, the usual max
					log.Printf("WARNING, timeout waiting for %d pending messages", len(p.pendings))
					break
				}
			}
			return
		}

		params := &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(p.url),
			MaxNumberOfMessages: aws.Int64(p.maxNumberOfMessages),
			WaitTimeSeconds:     aws.Int64(waitTimeSeconds),
			AttributeNames:      []*string{&MessageSystemAttributeNameSentTimestamp},
		}

		resp, err := p.svc.ReceiveMessage(params)

		if err != nil {
			log.Printf("ERROR: AWS session on %s - %s", p.url, err)
			time.Sleep(15 * time.Second)
			continue
		}

		for _, msg := range resp.Messages {
			p.deliver(msg)
		}
	}
}

func (p *sqsListen) deliver(msg *sqs.Message) {
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
		URL:           aws.String(p.url),
		SentTimestamp: sentTimestamp,
		Hash:          *msg.MessageId,
	}

	jsonParsed, err := gabs.ParseJSON(m.B)
	if err == nil && jsonParsed.Exists("Message") {
		s := strings.Replace(jsonParsed.S("Message").String(), "\\\"", "\"", -1)
		s = strings.TrimPrefix(s, "\"")
		s = strings.TrimSuffix(s, "\"")
		m.B = []byte(s)
	}

	if p.noBlocking {
		atLeastOneValid = p.deliverNoBlocking(m)
	} else {
		atLeastOneValid = p.deliverBlocking(m)
	}

	// We delete this message if is invalid for all the reactors
	if !atLeastOneValid {
		log.Printf("Invalid message from %s, deleted: %s", p.url, *msg.Body)
		p.delete(m)
	}

}

// deliverBlocking send the message to the reactors in sequence
func (p *sqsListen) deliverBlocking(m *Msg) (atLeastOneValid bool) {
	p.broadcastCh.Range(func(k, v interface{}) bool {
		if err := k.(*reactor.Reactor).MatchConditions(m); err != nil {
			return true
		}
		atLeastOneValid = true
		p.addPending(m)
		k.(*reactor.Reactor).Ch <- m
		return true
	})
	return
}

// deliverNoBlocking send the message in parallel to avoid blocking
// all messages due to a long standing reactor that has its chan full
func (p *sqsListen) deliverNoBlocking(m *Msg) (atLeastOneValid bool) {
	p.broadcastCh.Range(func(k, v interface{}) bool {
		if err := k.(*reactor.Reactor).MatchConditions(m); err != nil {
			return true
		}
		atLeastOneValid = true
		p.addPending(m)
		p.maxQueuedMessages.Access() // Check the limit of goroutines
		go func(m *Msg) {
			defer func() {
				p.maxQueuedMessages.Release()
				if r := recover(); r != nil {
					return // Ignore "closed channel" error when the program finishes
				}
			}()
			k.(*reactor.Reactor).Ch <- m
		}(m)
		return true
	})
	return
}

func (p *sqsListen) Stop() {
	if atomic.LoadUint32(&p.exiting) > 0 {
		return
	}
	atomic.AddUint32(&p.exiting, 1)
	log.Printf("SQS Input Stopping %s", p.url)
}

func (p *sqsListen) Exit() error {
	p.exitedMu.Lock()
	defer p.exitedMu.Unlock()

	if p.exited {
		return nil
	}

	p.Stop()
	p.exited = <-p.done
	return nil
}

func (p *sqsListen) delete(v lib.Msg) (err error) {
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

func (p *sqsListen) KeepAlive(ctx context.Context, t time.Duration, v lib.Msg) (err error) {
	msg, ok := v.(*Msg)
	if !ok {
		log.Printf("ERROR SQS KeepAlive: invalid message %+v", v)
		return
	}

	if msg.SQS == nil || msg == nil {
		return
	}

	// increase the visibility timeout by 10% of the original time
	t = t + time.Duration(float64(t)*0.1)
	sec := int64(math.Ceil(t.Seconds()))

	if _, err = msg.SQS.ChangeMessageVisibilityWithContext(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          msg.URL,
		ReceiptHandle:     msg.M.ReceiptHandle,
		VisibilityTimeout: aws.Int64(sec),
	}); err != nil {
		log.Printf("ERROR: %s - %s", *msg.URL, err)
	}
	return
}

func (p *sqsListen) addPending(m lib.Msg) {
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

// Done removes the message from the pending queue.
func (p *sqsListen) Done(m lib.Msg, statusOk bool) {
	msg, ok := m.(*Msg)
	if !ok {
		return
	}
	id := *msg.M.ReceiptHandle

	p.Lock()
	defer p.Unlock()

	// If it's not in pending, ignore it
	v, ok := p.pendings[id]
	if !ok {
		return
	}
	v -= 1
	p.pendings[id] = v
	if !statusOk {
		p.messError[id] = true
	}

	// Check if it's the last
	if v <= 0 {
		delete(p.pendings, id)
		_, hadError := p.messError[id]
		if !hadError {
			// Delete the message if there's no more pending reactors
			go p.delete(m) // Execute delete message outside the Lock
		}
	}
}
