# FIFO to Kinesis Pipeline

[![Build Status](https://travis-ci.org/acquia/fifo2kinesis.svg?branch=master)](https://travis-ci.org/acquia/fifo2kinesis)
[![Go Report Card](https://goreportcard.com/badge/github.com/acquia/fifo2kinesis)](https://goreportcard.com/report/github.com/acquia/fifo2kinesis)

This app continuously reads data from a [named pipe (FIFO)](https://en.wikipedia.org/wiki/Named_pipe)
and publishes it to a [Kinesis](https://aws.amazon.com/kinesis/) stream.

![fifo2kinesis cli demo](/doc/images/fifo2kinesis.gif)

## Why?

FIFOs are a great way to send data from one application to another. Having
an open pipe that ships data to Kinesis facilitates a lot of interesting use
cases. One such example is using the named pipe support in
[rsyslog](http://www.rsyslog.com/doc/v8-stable/configuration/modules/ompipe.html)
and [syslog-ng](https://www.balabit.com/sites/default/files/documents/syslog-ng-ose-latest-guides/en/syslog-ng-ose-guide-admin/html/configuring-destinations-pipe.html)
to send log streams to Kinesis.

Admittedly, it would be really easy to write a handful of lines of code in
a bash script using the AWS CLI to achieve the same result, however the
fifo2kinesis app is designed to reliably handle large volumes of data. It
achieves this by making good use of Go's concurrency primitives, buffering
and batch publishing data read from the fifo, and handling failures in a
way that can tolerate network and AWS outages.

## Installation

Either download the [latest binary](https://github.com/acquia/fifo2kinesis/releases/latest)
for your platform, or run the following command in the project's root to build
the aws-proxy binary from source:

```shell
GOPATH=$PWD go build -o ./bin/fifo2kinesis fifo2kinesis
```

## Usage

Create a named pipe:

```shell
mkfifo ./kinesis.pipe
```

Run the app:

```shell
./bin/fifo2kinesis --fifo-name=$(pwd)/kinesis.pipe --stream-name=my-stream
```

Write to the FIFO:

```shell
echo "Streamed at $(date)" > kinesis.pipe
```

The line will be published to the `my-stream` Kinesis stream within the
default flush interval of 5 seconds.

#### Quick start for the impatient among us

If you are impatient like me and want your oompa loompa now, modify the
`--buffer-queue-limit`, `--flush-interval`, and `--flush-handler` options so
that what you send to the FIFO is written to STDOUT immediately instead of a
buffered write to Kinesis. This doesn't do much, but it provides immediate
gratification and shows how the app works when you play with the options.

```shell
./bin/fifo2kinesis --fifo-name=$(pwd)/kinesis.pipe --buffer-queue-limit=1 --flush-interval=0 --flush-handler=logger
```

### Configuration

Configuration is read from command line options and environment variables
in that order of precedence. The following options and env variables are
available:

* `--fifo-name`, `FIFO2KINESIS_FIFO_NAME`: The absolute path of the named pipe.
* `--stream-name`, `FIFO2KINESIS_STREAM_NAME`: The name of the Kinesis stream.
* `--partition-key`, `FIFO2KINESIS_PARTITION_KEY`: The partition key, a random string if omitted.
* `--buffer-queue-limit`, `FIFO2KINESIS_BUFFER_QUEUE_LIMIT`: The number of items that trigger a buffer flush.
* `--failed-attempts-dir`, `FIFO2KINESIS_FAILED_ATTEMPTS_DIR`: The directory that logs failed attempts for retry.
* `--flush-interval`, `FIFO2KINESIS_FLUSH_INTERVAL`: The number of seconds before the buffer is flushed.
* `--flush-handler`, `FIFO2KINESIS_FLUSH_HANDLER`: Defaults to "kinesis", use "logger" for debugging.
* `--region`, `FIFO2KINESIS_REGION`: The AWS region that the Kinesis stream is provisioned in.
* `--role-arn`, `FIFO2KINESIS_ROLE_ARN`: The ARN of the AWS role being assumed.
* `--role-session-name`, `FIFO2KINESIS_ROLE_SESSION_NAME`: The session name used when assuming a role.
* `--debug`, `FIFO2KINESIS_DEBUG`: Show debug level log messages.

The application also requires credentials to publish to the specified
Kinesis stream. It uses the same [configuration mechanism](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html#config-settings-and-precedence)
as the AWS CLI tool, minus the command line options.

### Running With Upstart

Use [Upstart](http://upstart.ubuntu.com/) to start fifo2kinesis during boot
and supervise it while the system is running. Add a file to `/etc/init` with
the following contents, replacing `/path/to` and `my-stream` according to
your environment.

```
description "FIFO to Kinesis Pipeline"
start on runlevel [2345]

respawn
respawn limit 3 30
post-stop exec sleep 5

AWS_REGION=us-east-1
exec /path/to/fifo2kinesis --fifo-name=/path/to/named.pipe --stream-name=my-stream
```

### Publishing Logs From Syslog NG

**NOTE**: You might also want to check out [fluentd](http://www.fluentd.org/)
and the [Amazon Kinesis Agent](https://github.com/awslabs/amazon-kinesis-agent).
You won't find an argument in this README as to why you should choose one
over the other, I just want to make sure you have all the options in front
of you so that you can make the best decision for your specific use case.

Syslog NG provides the capability to use a named pipe as a destination. Use
fifo2kinesis to read log messages from the FIFO and publish them Kenisis.

Make a FIFO:

```
mkfifo /var/syslog.pipe
```

Modify syslog-ng configuration to send the logs to the named pipe. For example,
on Ubuntu 14.04 create a file named `/etc/syslog-ng/conf.d/01-kinesis.conf` with
the following configration:

```
destination d_pipe { pipe("/var/syslog.pipe"); };
log { source(s_src); destination(d_pipe); };
```

Start the app:

```
./bin/fifo2kinesis --fifo-name=/var/syslog.pipe --stream-name=my-stream
```

Restart syslog-ng:

```
service syslog-ng restart
```

The log stream will now be published to Kinesis.


## Development

AWS Proxy uses [Glide](https://glide.sh/) to manage dependencies.

#### Tests

Run the following commands to run tests and generate a coverage report:

```shell
GOPATH=$PWD go test -coverprofile=build/coverage.out fifo2kinesis
GOPATH=$PWD go tool cover -html=build/coverage.out
```