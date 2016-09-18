package main

import (
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"
)

// conf represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var conf *viper.Viper

// logger represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var logger *Logger

func main() {

	conf = viper.New()

	conf.SetEnvPrefix("FIFO2KINESIS")

	conf.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	conf.AutomaticEnv()

	viper.SetConfigName("fifo2kinesis")

	pflag.IntP("buffer-queue-limit", "l", 500, "The maximum number of items in the buffer before it is flushed")
	conf.BindPFlag("buffer-queue-limit", pflag.Lookup("buffer-queue-limit"))
	conf.SetDefault("buffer-queue-limit", 500)

	pflag.BoolP("debug", "d", false, "Show debug level log messages")
	conf.BindPFlag("debug", pflag.Lookup("debug"))
	conf.SetDefault("debug", "")

	pflag.StringP("failed-attempts-dir", "D", "", "The path to the directory containing failed attempts")
	conf.BindPFlag("failed-attempts-dir", pflag.Lookup("failed-attempts-dir"))
	conf.SetDefault("failed-attempts-dir", "")

	pflag.StringP("fifo-name", "f", "", "The absolute path of the named pipe, e.g. /var/test.pipe")
	conf.BindPFlag("fifo-name", pflag.Lookup("fifo-name"))
	conf.SetDefault("fifo-name", "")

	pflag.StringP("flush-handler", "h", "kinesis", "Defaults to \"kinesis\", use \"logger\" for debugging")
	conf.BindPFlag("flush-handler", pflag.Lookup("flush-handler"))
	conf.SetDefault("flush-handler", "kinesis")

	pflag.IntP("flush-interval", "i", 5, "The number of seconds before the buffer is flushed and written to Kinesis")
	conf.BindPFlag("flush-interval", pflag.Lookup("flush-interval"))
	conf.SetDefault("flush-interval", 5)

	pflag.StringP("partition-key", "p", "", "The partition key, defaults to a 12 character random string if omitted")
	conf.BindPFlag("partition-key", pflag.Lookup("partition-key"))
	conf.SetDefault("partition-key", "")

	pflag.StringP("role-arn", "r", "", "The ARN of the AWS role being assumed.")
	conf.BindPFlag("role-arn", pflag.Lookup("role-arn"))
	conf.SetDefault("partition-key", "")

	pflag.StringP("role-session-name", "S", "", "The session named used when assuming a role.")
	conf.BindPFlag("role-session-name", pflag.Lookup("role-session-name"))
	conf.SetDefault("role-session-name", "")

	pflag.StringP("stream-name", "s", "", "The name of the Kinesis stream")
	conf.BindPFlag("stream-name", pflag.Lookup("stream-name"))
	conf.SetDefault("stream-name", "")

	pflag.Parse()

	if conf.GetBool("debug") {
		logger = NewLogger(LOG_DEBUG)
	} else {
		logger = NewLogger(LOG_INFO)
	}

	logger.Debug("configuration parsed")

	h := conf.GetString("flush-handler")
	if h != "kinesis" && h != "logger" {
		logger.Fatalf("flush handler not valid: %s", h)
	}

	fn := conf.GetString("fifo-name")
	if fn == "" {
		logger.Fatal("missing required option: fifo-name")
	}

	sn := conf.GetString("stream-name")
	if h == "kinesis" && sn == "" {
		logger.Fatal("missing required option: stream-name")
	}

	ql := conf.GetInt("buffer-queue-limit")
	if ql < 1 {
		logger.Fatal("buffer queue limit must be greater than 0")
	} else if h == "kinesis" && ql > 500 {
		logger.Fatal("buffer queue cannot exceed 500 items when using the kinesis handler")
	}

	fifo := &Fifo{fn}

	bw := &MemoryBufferWriter{
		Fifo:          fifo,
		FlushInterval: conf.GetInt("flush-interval"),
		QueueLimit:    ql,
	}

	var bf BufferFlusher
	if h == "kinesis" {
		pk := conf.GetString("partition-key")
		bf = NewKinesisBufferFlusher(sn, pk)
	} else {
		bf = &LoggerBufferFlusher{}
	}

	var fh FailedAttemptHandler
	dir := conf.GetString("failed-attempts-dir")
	if dir == "" {
		fh = &NullFailedAttemptHandler{}
	} else {
		stat, err := os.Stat(dir)
		if os.IsNotExist(err) {
			logger.Fatal("failed attempts directory does not exist")
		} else if !stat.IsDir() {
			logger.Fatal("failed attempts directory is not a directory")
		} else if unix.Access(dir, unix.R_OK) != nil {
			logger.Fatal("failed attempts directory is not readable")
		} else {
			fh = &FileFailedAttemptHandler{dir, fifo}
		}
	}

	shutdown := EventListener()
	RunPipeline(fifo, &Buffer{bw, bf, fh}, shutdown)
}

// EventListener listens for SIGINT and SIGTERM signals and notifies the
// shutdown channel if it detects that either one was sent.
func EventListener() <-chan bool {
	shutdown := make(chan bool)

	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-ch:
				logger.Debug("shutdown signal received")
				shutdown <- true
				break
			}
		}
	}()

	return shutdown
}

// RunPipeline runs the pipeline that reads lines from the fifo, buffers the
// data, flushes the buffer (e.g. published the records to Kinesis), and
// saves failed requests for retry.
func RunPipeline(fifo *Fifo, buffer *Buffer, shutdown <-chan bool) {
	logger.Notice("starting pipeline")
	wg := &sync.WaitGroup{}

	// Ths code follows the pipeline pattern.
	// https://blog.golang.org/pipelines
	lines := ReadLines(fifo, wg)
	chunks := WriteToBuffer(lines, buffer)
	failed := FlushBuffer(chunks, buffer, wg)
	HandleFailures(failed, buffer, wg)

	RetryFailedAttempts(buffer)

	<-shutdown
	logger.Notice("stopping pipeline")

	fifo.SendCommand("stop")
	wg.Wait()

	logger.Notice("pipeline stopped")
}

// ReadLines reads lines from the fifo until a notification is sent to the
// stop channel. This is the source of the pipeline.
func ReadLines(fifo *Fifo, wg *sync.WaitGroup) <-chan []byte {
	lines := make(chan []byte)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(lines)
		if err := fifo.Scan(lines); err != nil {
			if perr, ok := err.(*os.PathError); ok {
				logger.Crit(perr.Error())
			} else {
				logger.Crit("error reading from fifo: %s", err)
			}
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}
	}()

	return lines
}

// WriteToBuffer fills the buffer with lines and turns them into groups of
// records that are send to the flush handler, e.g. Kinesis.
func WriteToBuffer(lines <-chan []byte, buffer *Buffer) <-chan [][]byte {

	// TODO Implement a buffer size limit
	// https://github.com/acquia/fifo2kinesis/issues/23
	chunks := make(chan [][]byte, 100)

	go func() {
		defer close(chunks)
		buffer.Write(lines, chunks)
	}()

	return chunks
}

// FlushBuffer batch-processes the lines that were read from the FIFO, e.g.
// issues a PutRecords command to the Kinesis stream.
func FlushBuffer(chunks <-chan [][]byte, buffer *Buffer, wg *sync.WaitGroup) <-chan [][]byte {
	failed := make(chan [][]byte)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(failed)
		buffer.Flush(chunks, failed)
	}()

	return failed
}

// HandleFailures saves failed chunks so that processing can be retried.
// This function is the pipeline's sink.
func HandleFailures(failed <-chan [][]byte, buffer *Buffer, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for attempt := range failed {
			logger.Debug("save failed attempt")
			if err := buffer.SaveAttempt(attempt); err != nil {
				logger.Error("%s", err)
			}

		}
	}()
}

// RetryFailedAttempts retries the failed attempts that were saved in the
// HandleFailures function.
func RetryFailedAttempts(buffer *Buffer) {
	go func() {
		for {
			// TODO Make the retry interval configurable
			// https://github.com/acquia/fifo2kinesis/issues/19
			time.Sleep(time.Second * 30)
			logger.Debug("retry failed attempts")
			buffer.Retry()
		}
	}()
}
