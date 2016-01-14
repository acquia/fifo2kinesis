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
fifo2kinesis --fifo-name=$(pwd)/test.pipe --stream-name=mystream
```

Write to the FIFO:

```shell
echo "Streamed at $(date)" > test.pipe
```

The line will be published to the `mystream` Kinesis stream.

### Enviornment Variables

The following environment variables are supported:

* `FIFO2KINESIS_FIFO_NAME`
* `FIFO2KINESIS_STREAM_NAME`
* `FIFO2KINESIS_DEBUG`

### Integrating With Syslog NG

Syslog NG provides the capability to use a named pipe as a destination. Use
this app to read log messages from the FIFO and publish them to a Kenisis
stream.

On Ubuntu 14.04, create a file named `/etc/syslog-ng/conf.d/01-kinesis.conf`
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
fifo2kinesis --fifo-name=/var/syslog.pipe --stream-name=mystream
```

Restart syslog-ng:

```
service syslog-ng restart
```

All log messages will be published to Kinesis.
