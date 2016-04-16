package main

import (
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// conf represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var conf *viper.Viper

// logger represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var logger *Logger

// init initializes the configuration and logging.
func init() {

	conf = viper.New()

	conf.SetEnvPrefix("FIFO2KINESIS")
	conf.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	conf.AutomaticEnv()

	viper.SetConfigName("fifo2kinesis")

	pflag.IntP("buffer-queue-limit", "l", 1, "The maximum number of items in the buffer before it is flushed")
	conf.BindPFlag("buffer-queue-limit", pflag.Lookup("buffer-queue-limit"))
	conf.SetDefault("buffer-queue-limit", 1)

	pflag.BoolP("debug", "d", false, "Show debug level log messages")
	conf.BindPFlag("debug", pflag.Lookup("debug"))
	conf.SetDefault("debug", "")

	pflag.StringP("fifo-name", "f", "", "The absolute path of the named pipe, e.g. /var/test.pipe")
	conf.BindPFlag("fifo-name", pflag.Lookup("fifo-name"))
	conf.SetDefault("fifo-name", "")

	pflag.StringP("flush-handler", "h", "kinesis", "Defaults to \"kinesis\", use \"logger\" for debugging")
	conf.BindPFlag("flush-handler", pflag.Lookup("flush-handler"))
	conf.SetDefault("flush-handler", "kinesis")

	pflag.IntP("flush-interval", "i", 0, "The number of seconds before the buffer is flushed and written to Kinesis")
	conf.BindPFlag("flush-interval", pflag.Lookup("flush-interval"))
	conf.SetDefault("flush-interval", 0)

	pflag.StringP("partition-key", "p", "", "The partition key, defaults to a 12 character random string if omitted")
	conf.BindPFlag("partition-key", pflag.Lookup("partition-key"))
	conf.SetDefault("partition-key", "")

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
}

func main() {

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

	fifo := &Fifo{fn}

	bw := &StandardBufferWriter{
		Fifo:          fifo,
		FlushInterval: conf.GetInt("flush-interval"),
		QueueLimit:    conf.GetInt("buffer-queue-limit"),
	}

	var bf BufferFlusher
	if h == "kinesis" {
		pk := conf.GetString("partition-key")
		bf = NewKinesisBufferFlusher(sn, pk)
	} else {
		bf = &LoggerBufferFlusher{}
	}

	shutdown := EventListener()
	RunPipeline(fifo, &Buffer{bw, bf}, shutdown)
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
// data, and writes records to Kinesis.
func RunPipeline(fifo *Fifo, buffer *Buffer, shutdown <-chan bool) {
	logger.Notice("starting pipeline")
	wg := &sync.WaitGroup{}

	lines := ReadLines(fifo, wg)
	chunks := WriteToBuffer(lines, buffer)
	FlushBuffer(chunks, buffer, wg)

	<-shutdown
	logger.Notice("stopping pipeline")

	fifo.SendCommand("stop")
	wg.Wait()

	logger.Notice("pipeline stopped")
}

// ReadLines reads lines from the fifo until a notification is sent to the
// stop channel. This is the source of the pipeline.
func ReadLines(fifo *Fifo, wg *sync.WaitGroup) <-chan string {
	lines := make(chan string)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(lines)
		if err := fifo.Scan(lines); err != nil {
			logger.Crit("error reading from fifo: ", err)
		}
	}()

	return lines
}

// WriteToBuffer fills the buffer with lines and turns them into groups of
// records that are published to Kinesis.
func WriteToBuffer(lines <-chan string, buffer *Buffer) <-chan []string {

	// TODO Figure out how many chunks we are willing to hold in memory.
	// We need a buffered channel so that ReadLines() is not blocked waiting
	// for buffer to be available. The max number of messages stored in
	// memory before blocking happens is 100 * the queue limit.
	chunks := make(chan []string, 100)

	go func() {
		defer close(chunks)
		buffer.Write(lines, chunks)
	}()

	return chunks
}

func FlushBuffer(chunks <-chan []string, buffer *Buffer, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		buffer.Flush(chunks)
	}()
}
