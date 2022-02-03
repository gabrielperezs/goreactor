[![Build Status](https://travis-ci.org/gabrielperezs/goreactor.svg?branch=master)](https://travis-ci.org/gabrielperezs/goreactor) [![Go Report Card](https://goreportcard.com/badge/github.com/gabrielperezs/goreactor)](https://goreportcard.com/report/github.com/gabrielperezs/goreactor)
![Go](https://github.com/gabrielperezs/goreactor/workflows/Go/badge.svg?branch=master)

Goreactor, trigger a message and execute commands
=================================================

The current version just support inputs from AWS SQS, in next versions will support also HTTP/S and Redis QUEUES.

Features
========

Substitutions in args entry in the config file
----------------------------------------------

- `$.path.with[0].jq.format`

    jq like format after `$.` _See [Usage case, message from SQS](#usage-case-message-from-sqs) for an example_

- `$.path.to.array...`

    If the jq like expression (Starting with `$.`) is the only text in the argument and ends with `...` _See [examples/ARRAY.md](examples/ARRAY.md) for an example_

    will expand the array in the specified path in different arguments.

    The use of this will replace the argument with N arguments where N is the number of elements in the array.

    NOTE: Should be the only text in the given argument element on the array.

- `${variableName}`

    Variables will be substituted in the arguments if the variable is in the following list. (This can also come from an expanded array)

    NOTE: Variables are case sensitive and curly brackets are mandatory

    Currently, we have the following variables:
    - `CreationTimestampMilliseconds` is the message creation time with in milliseconds
    - `CreationTimestampSeconds` is the message creation time in seconds _See [examples/ARRAY.md](examples/ARRAY.md) for an example_

Log outputs to stdout or firehose
----------------------------------

You can set logging to stdout with
```toml
[logstream]
logstream = "stdout"
```

You can set logging to firehose with
```toml
[logstream]
logstream = "firehose"
streamname = "example-goreactor-logs"
region = "eu-west-1"
```

You can disable logging if you leave logstream empty or set to none.
Or not defining logstream at all
```toml
[logstream]
logstream = ""
```
```toml
[logstream]
logstream = "none"
```


The output will be a json per output line with the following format:
```json
{"Host":"RUNNER_HOSTNAME","Pid":44274,"RID":1,"TID":1,"Line":0,"Output":"./print_some_lines_and_exit ","Status":"CMD","Timestamp":1635149955}
{"Host":"RUNNER_HOSTNAME","Pid":44274,"RID":1,"TID":1,"Line":1,"Output":"first line","Status":"RUN","Timestamp":1635149955}
{"Host":"RUNNER_HOSTNAME","Pid":44274,"RID":1,"TID":1,"Line":2,"Output":"second line","Status":"RUN","Timestamp":1635149955}
{"Host":"RUNNER_HOSTNAME","Pid":44274,"RID":1,"TID":1,"Line":3,"Output":"last line","Status":"RUN","Timestamp":1635149955}
{"Host":"RUNNER_HOSTNAME","Pid":44274,"RID":1,"TID":1,"Line":4,"Status":"END","Elapse":3.067383247,"Timestamp":1635149955}
```


Set working directory
---------------------

The working directory of the process can be set with `workingDirectory`.

Execute as a specific user
--------------------------

It may be desired to execute some commands as a specific user.

NOTE: that this only works in unix or unix like systems and have only been tested in linux. In other OS, this will be ignored.

In order to do so, each [[reactor]] entry may have a `user` value defined with the desired user.
The env will be empty (only containing the HOME - which can be overridden) unless you specify it in a `env` section
that will only be used if `user` is defined.

```toml
[[reactor]]
# (...) All the desired values
user = "john"
env = [
"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
"AWS_PROFILE=stage"
]
```

This would run the reactor using john user and 2 environ: HOME set to the home defined for the john user,
PATH defined to "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" and AWS_PROFILE set to "stage"

**DO NOT USE SET THE SETUID BIT FOR GOREACTOR!!!!**

Usage case, message from SQS
===========================

In the next example, we will run a command after receiving a message from SQS, the message received will look like the JSON below

Message received from SQS
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
workingDirectory = "/path/to/process/wd/"
```


The daemon will execute a command like
--------------------------------------

```bash
/usr/local/bin/do-something-with-the instance asg=SOMEGROUPNAME instance_id=i-00000009999
```
