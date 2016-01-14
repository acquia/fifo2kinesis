package main

import (
	"bufio"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

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

// RunPipeline reads lines from the named pipe and publishes data records to
// Kinesis. It opens "fifo-name" and publishes records to the "stream-name"
// Kinesis stream.
func (fifo *Fifo) RunPipeline() error {

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

	return nil
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
