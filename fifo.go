package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"

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
func (fifo *Fifo) RunPipeline(wg *sync.WaitGroup) (err error) {
	for {
		if err = fifo.ReadFifo(wg); err != nil {
			return
		}
	}
}

// ReadFifo reads a line from the fifo and publishes the data record.
func (fifo *Fifo) ReadFifo(wg *sync.WaitGroup) (err error) {
	var file *os.File

	logger.Debug("reading data from fifo: %s", fifo.name)
	file, err = os.OpenFile(fifo.name, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {

		// Still a tiny race condition here. Better than it was, but could
		// be improved since we could lose a message during shutdown. Add a
		// sleep above this line to prove the race condition.
		// https://github.com/acquia/fifo2kinesis/issues/9
		wg.Add(1)
		defer wg.Done()

		data := scanner.Bytes()
		logger.Debug("line read from fifo: %s", data)
		if err = fifo.PublishDataRecord(data); err != nil {
			return
		}
	}

	return
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
		logger.Debug("data record published: sequence-number=%s partition-key=%s", *output.SequenceNumber, *params.PartitionKey)
	} else {
		return fmt.Errorf("error publishing data record: %s", err)
	}

	return nil
}
