package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

// KinesisBufferFlusher implements BufferFlusher and publishes the records
// that are emitted by the BufferWriter to a Kinesis stream.
//
// Name is the Kinesis stream name.
//
// PartitionKey is the partition key set for all records. If PartitionKey is
// an empty string, then a random string is generated for all data records
// which is useful for distributing records across all open shards.
//
// kinesis is the initialized Kinesis client.
type KinesisBufferFlusher struct {
	Name         *string
	PartitionKey string
	kinesis      *kinesis.Kinesis
}

// NewKinesisBufferFlusher returns a KinesisBufferFlusher configured with
// the stream name and partition key.
func NewKinesisBufferFlusher(name, partitionKey string) *KinesisBufferFlusher {
	return &KinesisBufferFlusher{
		Name:         aws.String(name),
		PartitionKey: partitionKey,
		kinesis:      kinesis.New(session.New()),
	}
}

// FormatPartitionKey either returns the configured partition key or a
// random string of 12 characters if the partition key is an empty string.
func (f *KinesisBufferFlusher) FormatPartitionKey() *string {
	if f.PartitionKey == "" {
		return aws.String(RandomString(12))
	} else {
		return aws.String(f.PartitionKey)
	}
}

// Flush publishes the data consumed from chunks to a Kenisis stream and
// emits failed records to the failed channel.
func (f *KinesisBufferFlusher) Flush(chunks <-chan []string, failed chan []string) {
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

		// Check if all the records failed to be published.
		output, err := f.kinesis.PutRecords(params)
		if err != nil {
			logger.Error("error publishing record(s) to kinesis: %s", err)
			failed <- chunk
			continue
		}

		// Check if some of the records failed to be published.
		if *output.FailedRecordCount != 0 {
			logger.Error("error publishing %v record(s) to kinesis: %s", *output.FailedRecordCount, err)
			subchunk := make([]string, *output.FailedRecordCount)

			for key, record := range output.Records {
				if record.ErrorCode != nil {
					subchunk[key] = chunk[key]
				}
			}

			failed <- subchunk
		}

		total := int64(size) - *output.FailedRecordCount
		if total != 0 {
			logger.Debug("published %v record(s) to kinesis", total)
		}
	}
}
