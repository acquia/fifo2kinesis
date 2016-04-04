package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

// Fifo represents the named pipe.
type Fifo struct {
	kinesis      *kinesis.Kinesis
	name         string
	stream       *string
	partitionKey string
}

// NewFifo creates a new instance of the Fifo struct.
func NewFifo(fifoName, streamName, partitionKey string) *Fifo {
	return &Fifo{
		kinesis:      kinesis.New(session.New()),
		name:         fifoName,
		stream:       aws.String(streamName),
		partitionKey: partitionKey,
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
		if err := fifo.PublishDataRecord(data); err != nil {
			return err
		}
	}

	return nil
}

// PublishDataRecord publishes individual data records to a Kinesis stream.
func (fifo *Fifo) PublishDataRecord(data []byte) error {

	params := &kinesis.PutRecordInput{
		Data:       data,
		StreamName: fifo.stream,
	}

	// Default to a 12 character random key if no partition key is set.
	if fifo.partitionKey == "" {
		params.PartitionKey = aws.String(RandomString(12))
	} else {
		params.PartitionKey = aws.String(fifo.partitionKey)
	}

	if output, err := fifo.kinesis.PutRecord(params); err == nil {
		logger.Debug("data record published with sequence number: %s", *output.SequenceNumber)
	} else {
		return fmt.Errorf("error publishing data record: %s", err)
	}

	return nil
}
