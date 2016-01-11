package main

import (
	"bufio"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/spf13/viper"
)

// conf represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var conf *viper.Viper

// init initializes the configuration.
func init() {
	conf = viper.New()

	viper.SetConfigName("fifo2kinesis")
	viper.AddConfigPath("$HOME/.config")

	conf.SetDefault("stream-name", "")
	conf.BindEnv("stream-name", "FIFO2KINESIS_STREAM_NAME")

	conf.SetDefault("fifo-name", "")
	conf.BindEnv("fifo-name", "FIFO2KINESIS_FIFO_NAME")
}

func main() {

	fn := conf.GetString("fifo-name")
	if fn == "" {
		log.Fatal("missing required option: fifo-name")
	}

	sn := conf.GetString("stream-name")
	if sn == "" {
		log.Fatal("missing required option: stack-name")
	}

	fifo := NewFifo(fn, sn)

	err := fifo.StartPipeline()
	if err != nil {
		log.Fatal(err)
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

// StartPipeline continuously read data from the named pipe and sends
// records to Kinesis. It opens "fifo-name" and publishes records to the
// "stream-name" Kinesis stream.
func (fifo *Fifo) StartPipeline() error {
	for {

		file, err := os.OpenFile(fifo.name, os.O_RDONLY, os.ModeNamedPipe)
		if err != nil {
			return err
		}

		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		// @todo batch processing and error handling.
		for scanner.Scan() {
			data := scanner.Bytes()
			fifo.PutRecord(data)
		}
	}
}

// Publishes records to the Kinesis stream.
func (fifo *Fifo) PutRecord(data []byte) {

	params := &kinesis.PutRecordInput{
		Data:         data,
		PartitionKey: aws.String("PartitionKey"), // @todo change this
		StreamName:   fifo.stream,
	}

	_, _ = fifo.kinesis.PutRecord(params)
}
