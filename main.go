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
	"github.com/spf13/viper"
)

// conf represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var conf *viper.Viper

// logger represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var logger *Logger

// init initializes the configuration.
func init() {

	conf = viper.New()

	conf.SetEnvPrefix("FIFO2KINESIS")
	conf.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	conf.AutomaticEnv()

	viper.SetConfigName("fifo2kinesis")

	conf.SetDefault("fifo-name", "")
	conf.SetDefault("stream-name", "")

	logger = NewLogger(LOG_DEBUG)
	logger.Debug("initialized configuration")
}

func main() {

	fn := conf.GetString("fifo-name")
	if fn == "" {
		logger.Fatal("missing required option: fifo-name")
	}

	sn := conf.GetString("stream-name")
	if sn == "" {
		logger.Fatal("missing required option: stack-name")
	}

	StartPipeline(NewFifo(fn, sn))
	logger.Notice("pipeline stopped")
}

func StartPipeline(fifo *Fifo) {
	logger.Notice("starting pipeline")

	wg := &sync.WaitGroup{}
	stop := make(chan bool)

	go EventListener(wg, stop)
	fifo.RunPipeline(wg, stop)
}

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

// RunPipeline read sdata from the named pipe and publishes data records to
// Kinesis. It opens "fifo-name" and publishes records to the "stream-name"
// Kinesis stream.
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
				logger.Debug("read line from fifo: %s", data)
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
		logger.Debug("published data record with sequence number: %s", *output.SequenceNumber)
	} else {
		logger.Error("error publishing data record: %s", err)
	}
}
