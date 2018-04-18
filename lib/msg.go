package lib

import "github.com/aws/aws-sdk-go/service/sqs"

type Msg struct {
	URL *string
	SQS *sqs.SQS
	M   *sqs.DeleteMessageBatchRequestEntry
	B   []byte
}
