package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

type KinesisBufferFlusher struct {
	Name         *string
	PartitionKey string
	kinesis      *kinesis.Kinesis
}

func NewKinesisBufferFlusher(name, partitionKey string) *KinesisBufferFlusher {
	return &KinesisBufferFlusher{
		Name:         aws.String(name),
		PartitionKey: partitionKey,
		kinesis:      kinesis.New(session.New()),
	}
}

func (f *KinesisBufferFlusher) FormatPartitionKey() *string {
	if f.PartitionKey == "" {
		return aws.String(RandomString(12))
	} else {
		return aws.String(f.PartitionKey)
	}
}

func (f *KinesisBufferFlusher) Flush(chunks <-chan []string) {
	for chunk := range chunks {
		// TODO Create the PutRecords command for Kinesis. We are just
		// printing this for debugging purposes.
		for _, line := range chunk {

			params := &kinesis.PutRecordInput{
				Data:         []byte(line),
				StreamName:   f.Name,
				PartitionKey: f.FormatPartitionKey(),
			}

			if output, err := f.kinesis.PutRecord(params); err == nil {
				logger.Debug("data record published: sequence-number=%s partition-key=%s", *output.SequenceNumber, *params.PartitionKey)
			} else {
				logger.Error("error publishing data record: %s", err)
			}
		}
	}
}
