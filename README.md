# FIFO to Kinesis Pipeline

This app continuously reads data from a named pipe (FIFO) and publishes it
to a Kinesis stream.

## Usage

Create a named pipe:

```shell
mkfifo ./test.pipe
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

Syslog NG provides the capability to use a named pipe as a destination. Use
this app to read log messages from the FIFO and publish them to a Kenisis
stream.

Ubuntu 14.04, create a file named `/etc/syslog-ng/conf.d/01-kinesis.conf`
with the following configration:

```
destination d_pipe { pipe("/var/syslog.pipe"); };
log { source(s_src); destination(d_pipe); };
```

Make a FIFO:

```
mkfifo /var/syslog.pipe
```

Start the app:

```
FIFO2KINESIS_FIFO_NAME=/var/syslog.pipe FIFO2KINESIS_STREAM_NAME=mystream fifo2kinesis
```

Restart syslog-ng:

```
service restart syslog-ng
```

All log messages will be published to Kinesis.
