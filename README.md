# FIFO to Kinesis Pipeline

Reads data from a named pipe (FIFO) and published it to a Kinesis stream.

## Usage

Create a named pipe:

```shell
mkfifo test.pipe
```

Run the app:

```shel
FIFO2KINESIS_FIFO_NAME=$(pwd)/test.pipe FIFO2KINESIS_STREAM_NAME=mystream fifo2kinesis
```

Write to the FIFO:

```shell
echo "Streamed at $(date)" > test.pipe
```

The line will be published to the `mystream` Kinesis stream.

### Integrating With Syslog NG

Syslog NG provides the capability to use a named pipe as a destination. On
Ubuntu 14.04, add the following configuration to write all log messages to
the `/var/syslog.pipe` FIFO:

```
destination d_pipe { pipe("/var/syslog.pipe"); };
log { source(s_src); destination(d_pipe); };
```

Use this app to read from the FIFO and publish log messages to Kinesis.
