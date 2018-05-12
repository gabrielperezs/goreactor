[![Build Status](https://travis-ci.org/gabrielperezs/goreactor.svg?branch=master)](https://travis-ci.org/gabrielperezs/goreactor) [![Go Report Card](https://goreportcard.com/badge/github.com/gabrielperezs/goreactor)](https://goreportcard.com/report/github.com/gabrielperezs/goreactor)

Goreactor, trigger a message and execute commands
=================================================

The current version just support inputs from AWS SQS, in next versions will support also HTTP/S and Redis QUEUES.


Usage case, message from SQS
===========================

In the next example, we will run a command after receiving a message from SQS, the message received will look like the JSON below

Message recived from SQS
------------------------

```json
{
    "Progress": 50,
    "AccountId": "9999999999",
    "Description": "Launching EC2 instance: i-00000009999",
    "RequestId": "5f88cdf3-311c-4d61-bdec-0000000000",
    "EndTime": "2018-04-18T14:17:22.779Z",
    "AutoScalingGroupARN": "arn:aws:autoscaling:eu-west-1:9999999999:autoScalingGroup:0b35d38a-8270-45d0-a0d8-0000000000:autoScalingGroupName/SOMEGROUPNAME",
    "ActivityId": "5f88cdf3-311c-4d61-bdec-0000000000",
    "StartTime": "2018-04-18T14:16:26.804Z",
    "Service": "AWS Auto Scaling",
    "Time": "2018-04-18T14:17:22.779Z",
    "EC2InstanceId": "i-00000009999", // <--- We read this
    "StatusCode": "InProgress",
    "StatusMessage": "",
    "Details": {
        "Subnet ID": "subnet-XXXXXXXX",
        "Availability Zone": "eu-west-1a"
    },
    "AutoScalingGroupName": "SOMEGROUPNAME", // <--- We read this
    "Cause": "At 2018-04-18T14:16:21Z a user request update of AutoScalingGroup constraints to min: 1, max: 5, desired: 3 changing the desired capacity from 2 to 3....",
    "Event": "autoscaling:EC2_INSTANCE_LAUNCH" // <--- We read this, as condition
}
```

Configuration in go reactor
---------------------------

In the goreactor, we are ready to listen to messages and execute the command.

Please __put attention__ in the "cond" parameter (is a condition, like an "if"). __This means that the command will be executed JUST if we can find "Event = autoscaling:EC2_INSTANCE_LAUNCH" in the JSON__. If this key is not there or the value is not what we defined, then goreactor will ignore the message


```toml
logfile = "/var/log/goreactor.log"

[[reactor]]
concurrent = 10
delay = "5s"
input = "sqs"
url = "https://sqs.eu-west-1.amazonaws.com/9999999999/testing"
profile = "default"
region = "eu-west-1"
output = "cmd"
cond = [ 
    { "$.Event" = "autoscaling:EC2_INSTANCE_LAUNCH" }
]
cmd = "/usr/local/bin/do-something-with-the-instance"
args = ["asg=$.AutoScalingGroupName", "instance_id=$.EC2InstanceId"]
argsjson = true
```

The daemon will execute a command like
--------------------------------------

```bash
/usr/local/bin/do-something-with-the instance asg=SOMEGROUPNAME instance_id=i-00000009999
```