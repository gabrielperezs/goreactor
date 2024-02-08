Example with expanded array
===========================

In the next example, we will run a command after receiving a message from SQS, the message received will look like the JSON below

Received message
----------------

```json
{
  "args":["--timestamp", "${CreationTimestampSeconds}"]
}
```

Configuration in go reactor
---------------------------

In the goreactor, we are ready to listen to messages and execute the command.

```toml
[logstream]
logstream = "stdout"

[[reactor]]
concurrent = 10
delay = "5s"
input = "sqs"
url = "https://sqs.eu-west-1.amazonaws.com/9999999999/testing"
profile = "default"
region = "eu-west-1"
output = "cmd"
cmd = "/usr/local/bin/do-something-for-that-time"
args = ["--example-argument-before", "$.args...", "--example-argument-after", "--other-timestamp=${CreationTimestampSeconds}"]
```


The daemon will execute a command like
--------------------------------------

```bash
/usr/local/bin/do-something-for-that-time --example-argument-before --timestamp 1591877960 --example-argument-after --other-timestmp=1591877960
```
Note that ${CreationTimestampSeconds} from the message was overridden with 1591877960, 
that's the time when the message was created in the queue.
Same thing with --other-timestamp=1591877960.
