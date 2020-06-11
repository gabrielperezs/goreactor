package sqs

import (
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Msg is the message struct that were captured by the input plugin
type Msg struct {
	URL           *string
	SQS           *sqs.SQS
	M             *sqs.DeleteMessageBatchRequestEntry
	B             []byte
	SentTimestamp int64
}

// Body will return the bytes of the SQS message
func (m *Msg) Body() []byte {
	return m.B
}

// CreationTimestampMilliseconds will return the creation timestamp for the message
func (m *Msg) CreationTimestampMilliseconds() int64 {
	return m.SentTimestamp
}