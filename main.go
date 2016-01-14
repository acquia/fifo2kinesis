package main

import (
	"bufio"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
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

	pflag.String("fifo-name", "f", "The absolute path of the named pipe, e.g. /var/test.pipe")
	conf.BindPFlag("fifo-name", pflag.Lookup("fifo-name"))
	conf.SetDefault("fifo-name", "")

	pflag.StringP("stream-name", "s", "", "The name of the Kinesis stream")
	conf.BindPFlag("stream-name", pflag.Lookup("stream-name"))
	conf.SetDefault("stream-name", "")

	pflag.BoolP("debug", "d", false, "Show debug level log messages")
	conf.BindPFlag("debug", pflag.Lookup("debug"))
	conf.SetDefault("debug", "")

	pflag.Parse()

	if conf.GetBool("debug") {
		logger = NewLogger(LOG_DEBUG)
	} else {
		logger = NewLogger(LOG_INFO)
	}

	logger.Debug("configuration parsed")
}

func main() {

	fn := conf.GetString("fifo-name")
	if fn == "" {
		logger.Fatal("missing required option: fifo-name")
	}

	sn := conf.GetString("stream-name")
	if sn == "" {
		logger.Fatal("missing required option: stream-name")
	}

	StartPipeline(NewFifo(fn, sn))
	logger.Notice("pipeline stopped")
}

// StartPipeline sets up the event handler and starts the pipeline.
func StartPipeline(fifo *Fifo) {
	logger.Notice("starting pipeline")

	wg := &sync.WaitGroup{}
	stop := make(chan bool)

	go EventListener(wg, stop)
	fifo.RunPipeline(wg, stop)
}

// EventListener listens for signals in order to stop the application.
func EventListener(wg *sync.WaitGroup, stop chan bool) {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			logger.Notice("stopping pipeline")
			stop <- true
			break
		}
	}
}

// Fifo represents the named pipe.
type Fifo struct {
	kinesis *kinesis.Kinesis
	name    string
	stream  *string
}

// NewFifo creates a new instance of the Fifo struct.
func NewFifo(fifoName, streamName string) *Fifo {
	return &Fifo{
		kinesis: kinesis.New(session.New()),
		name:    fifoName,
		stream:  aws.String(streamName),
	}
}

// RunPipeline continuously reads data from the named pipe and publishes
// data records to Kinesis. It opens "fifo-name" and publishes records to
// the "stream-name" Kinesis stream.
func (fifo *Fifo) RunPipeline(wg *sync.WaitGroup, stop chan bool) error {
	for {
		select {
		case <-stop:
			wg.Wait()
			logger.Notice("stopping pipeline")
			return nil

		default:
			logger.Debug("reading data from fifo: %s", fifo.name)
			file, err := os.OpenFile(fifo.name, os.O_RDONLY, os.ModeNamedPipe)
			if err != nil {
				return err
			}

			defer file.Close()

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)

			for scanner.Scan() {
				data := scanner.Bytes()
				logger.Debug("line read from fifo: %s", data)
				fifo.PublishDataRecord(data)
			}
		}
	}
}

// Publishes individual data records to the Kinesis stream.
func (fifo *Fifo) PublishDataRecord(data []byte) {

	params := &kinesis.PutRecordInput{
		Data:         data,
		PartitionKey: aws.String("PartitionKey"), // @todo change this
		StreamName:   fifo.stream,
	}

	if output, err := fifo.kinesis.PutRecord(params); err == nil {
		logger.Debug("data record published with sequence number: %s", *output.SequenceNumber)
	} else {
		logger.Error("error publishing data record: %s", err)
	}
}
