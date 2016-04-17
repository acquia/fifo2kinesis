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
		size := len(chunk)
		
		if size < 1 {
			continue
		}

		records := make([]*kinesis.PutRecordsRequestEntry, size)
		for key, line := range chunk {
			records[key] = &kinesis.PutRecordsRequestEntry{
				PartitionKey: f.FormatPartitionKey(),
				Data:         []byte(line),
			}
		}

		params := &kinesis.PutRecordsInput{
			StreamName: f.Name,
			Records:    records,
		}

		if output, err := f.kinesis.PutRecords(params); err == nil {
			logger.Debug("%v record(s) published to kinesis", size)
		} else {
			logger.Error("error publishing %v data record(s): %s", output.FailedRecordCount, err)
		}
	}
}
