package cmd

import (
	"testing"

	"github.com/gabrielperezs/goreactor/lib"
	"github.com/gabrielperezs/goreactor/reactor"
	"github.com/stretchr/testify/assert"
)

type Msg struct {
	B    []byte
	ts   int64
	hash string
}

func (m *Msg) Body() []byte {
	return m.B
}

func (m *Msg) CreationTimestampMilliseconds() int64 {
	return m.ts
}

func (m *Msg) GetHash() string {
	return m.hash
}

func (m *Msg) Done() {
}

func (m *Msg) Wait() {
}

func TestJqReplaceActuallyReplacing(t *testing.T) {
	var r *reactor.Reactor = nil

	c := make(map[string]any)
	c["cmd"] = "cmd_name"
	c["args"] = []any{"$.lang", "$.script"}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", cmd.user)
	assert.Equal(t, "cmd_name", cmd.cmd)
	assert.Equal(t, 2, len(cmd.args))
	assert.Equal(t, "$.lang", cmd.args[0])
	assert.Equal(t, "$.script", cmd.args[1])

	var msg lib.Msg = &Msg{
		B:  []byte("{\"lang\":\"python3\",\"script\":\"script01\"}"),
		ts: 1591784694,
	}

	var args = cmd.getReplacedArguments(msg)

	assert.Equal(t, 2, len(args))
	assert.Equal(t, "python3", args[0])
	assert.Equal(t, "script01", args[1])
}

func TestFindReplaceReturningSlice(t *testing.T) {
	var r *reactor.Reactor = nil

	c := make(map[string]any)
	c["cmd"] = "cmd_name"
	c["args"] = []any{"$.lang", "$.script", "$.args..."}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", cmd.user)
	assert.Equal(t, "cmd_name", cmd.cmd)
	assert.Equal(t, 3, len(cmd.args))
	assert.Equal(t, "$.lang", cmd.args[0])
	assert.Equal(t, "$.script", cmd.args[1])
	assert.Equal(t, "$.args...", cmd.args[2])

	var msg lib.Msg = &Msg{
		B:  []byte(`{"lang":"python3","script":"script01","args":["third", "fourth"]}`),
		ts: 1591784694,
	}

	var args = cmd.getReplacedArguments(msg)

	assert.Equal(t, 4, len(args))
	assert.Equal(t, "python3", args[0])
	assert.Equal(t, "script01", args[1])
	assert.Equal(t, "third", args[2])
	assert.Equal(t, "fourth", args[3])
}

func TestFindReplaceS3Example(t *testing.T) {
	var r *reactor.Reactor = nil

	c := make(map[string]any)
	c["cmd"] = "/usr/local/bin/suppliers-metrics"
	c["args"] = []any{"-plugin=$.Records.[0].s3.bucket.name", "-file=$.Records.[0].s3.object.key", "-config=/usr/local/etc/suppliers-metrics.conf"}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	var msg lib.Msg = &Msg{
		B: []byte(`
{
   "Records":[
      {
         "eventVersion":"2.2",
         "eventSource":"aws:s3",
         "awsRegion":"us-west-2",
         "eventName":"event-type",
         "userIdentity":{
            "principalId":"Amazon-customer-ID-of-the-user-who-caused-the-event"
         },
         "requestParameters":{
            "sourceIPAddress":"ip-address-where-request-came-from"
         },
         "responseElements":{
            "x-amz-request-id":"Amazon S3 generated request ID",
            "x-amz-id-2":"Amazon S3 host that processed the request"
         },
         "s3":{
            "s3SchemaVersion":"1.0",
            "configurationId":"ID found in the bucket notification configuration",
            "bucket":{
               "name":"bucket-name",
               "ownerIdentity":{
                  "principalId":"Amazon-customer-ID-of-the-bucket-owner"
               },
               "arn":"bucket-ARN"
            },
            "object":{
               "key":"object-key",
               "eTag":"object eTag",
               "versionId":"object version if bucket is versioning-enabled, otherwise null",
               "sequencer": "a string representation of a hexadecimal value used to determine event sequence, only used with PUTs and DELETEs"
            }
         },
         "glacierEventData": {
            "restoreEventData": {
               "lifecycleRestorationExpiryTime": "The time, in ISO-8601 format, for example, 1970-01-01T00:00:00.000Z, of Restore Expiry",
               "lifecycleRestoreStorageClass": "Source storage class for restore"
            }
         }
      }
   ]
}`),
		ts: 1591784694,
	}

	var args = cmd.getReplacedArguments(msg)

	assert.Equal(t, 3, len(args))
	assert.Equal(t, `-plugin=bucket-name`, args[0])
	assert.Equal(t, `-file=object-key`, args[1])
	assert.Equal(t, `-config=/usr/local/etc/suppliers-metrics.conf`, args[2])
}

func TestFindReplacePassJsonItself(t *testing.T) {
	var r *reactor.Reactor = nil

	c := make(map[string]any)
	c["cmd"] = "cmd_name"
	c["args"] = []any{"$.."}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	var msg lib.Msg = &Msg{
		B:  []byte(`{"first_key": "value"}`),
		ts: 1591784694,
	}

	var args = cmd.getReplacedArguments(msg)

	assert.Equal(t, 1, len(args))
	assert.Equal(t, `{"first_key": "value"}`, args[0])
}

func TestFindReplaceExpandArray(t *testing.T) {
	var r *reactor.Reactor = nil

	c := make(map[string]any)
	c["cmd"] = "cmd_name"
	c["args"] = []any{"$...", "$...."} //Should test both just in case at some point it's used.

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	var msg lib.Msg = &Msg{
		B:  []byte(`["first", "second"]`),
		ts: 1591784694,
	}

	var args = cmd.getReplacedArguments(msg)

	assert.Equal(t, 4, len(args))
	assert.Equal(t, "first", args[0])
	assert.Equal(t, "second", args[1])
	assert.Equal(t, "first", args[2])
	assert.Equal(t, "second", args[3])
}

func TestFindReplaceVariables(t *testing.T) {
	var r *reactor.Reactor = nil

	c := make(map[string]any)
	c["cmd"] = "cmd_name"
	c["args"] = []any{"--timestamp", "${CreationTimestampMilliseconds}"} //Should test both just in case at some point it's used.

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	var msg lib.Msg = &Msg{
		B:  []byte(""),
		ts: 1591784694,
	}

	var args = cmd.getReplacedArguments(msg)

	assert.Equal(t, 2, len(args))
	assert.Equal(t, "--timestamp", args[0])
	assert.Equal(t, "1591784694", args[1])
}

func TestFindReplaceWithArrayAndVariable(t *testing.T) {
	var r *reactor.Reactor = nil

	c := make(map[string]any)
	c["cmd"] = "cmd_name"
	c["args"] = []any{"$.lang", "$.script", "$.args..."}

	cmd, err := NewOrGet(r, c)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", cmd.user)
	assert.Equal(t, "cmd_name", cmd.cmd)
	assert.Equal(t, 3, len(cmd.args))
	assert.Equal(t, "$.lang", cmd.args[0])
	assert.Equal(t, "$.script", cmd.args[1])
	assert.Equal(t, "$.args...", cmd.args[2])

	var msg lib.Msg = &Msg{
		B: []byte(`{"lang":"python3","script":"script01","args":
						["--timestamp", "${CreationTimestampSeconds}", 
							"--timestamp-with-milliseconds-precision", "${CreationTimestampMilliseconds}"]}`),
		ts: 1591784694,
	}

	var args = cmd.getReplacedArguments(msg)

	assert.Equal(t, 6, len(args))
	assert.Equal(t, "python3", args[0])
	assert.Equal(t, "script01", args[1])
	assert.Equal(t, "--timestamp", args[2])
	assert.Equal(t, "1591784", args[3])
	assert.Equal(t, "--timestamp-with-milliseconds-precision", args[4])
	assert.Equal(t, "1591784694", args[5])
}
