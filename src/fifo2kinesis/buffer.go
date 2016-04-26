package main

import "time"

// BufferWriter is implemented by subsystems that accept data read from the
// FIFO and groups it into chunks that are processed by the BufferFlusher.
//
// Write accepts data read from the FIFO through the lines channel and emits
// grouped chunks of data via the chunks channel that is intended to be read
// by the BufferFlusher's Flush method for processing.
type BufferWriter interface {
	Write(lines <-chan string, chunks chan []string)
}

// BufferFlusher is the interface implemented by subsystems that process the
// chunks of data passed to it by the BufferWriter. One example is the
// implementation that publishes the data records to Kinesis.
//
// Flush processes a chunk of data passed to it through the chunks channel
// by the buffer flusher. For example, the KinesisBufferFlusher batch
// publishes the chunk of records to a Kinesis stream.
type BufferFlusher interface {
	Flush(chunks <-chan []string, failed chan []string)
}

// FailedAttemptHandler is the interface implemented by subsystems that
// handle failed records that couldn't be processed by the BufferFlusher.
//
// SaveAttempt stores the failed lines passed to it through the attempt
// channel for retry at a later time.
//
// Retry handles the records that were queued for retry in the SaveAttempt
// method. Usually this means writing the lines back to the FIFO so they can
// go through the pipeline again.
type FailedAttemptHandler interface {
	SaveAttempt(attempt []string) error
	Retry()
}

// Buffer is the interface that groups the BufferWriter, BufferFlusher, and
// FailedAttemptHandler. In other words, it handles everything after data is
// read from the FIFO.
type Buffer struct {
	BufferWriter
	BufferFlusher
	FailedAttemptHandler
}

// MemoryBufferWriter implemnents BufferWriter and reads lines from the fifo
// into memory to group them into chunks prior to emitting them to the
// BufferFlusher. This BufferWriter is efficient, but if the application or
// system crashes then the data in the buffer is lost.
type MemoryBufferWriter struct {
	Fifo          *Fifo
	FlushInterval int
	QueueLimit    int
}

// reset is a helper method that returns an initialized chunk, key position,
// and flush flag.
func (w *MemoryBufferWriter) reset() ([]string, int, bool) {
	return make([]string, w.QueueLimit), 0, false
}

// Write stores the lines in memory that were read from the FIFO and emits
// them as chunks for processing by the BufferFlusher.
func (w *MemoryBufferWriter) Write(lines <-chan string, chunks chan []string) {
	forceFlush := make(chan bool, 1)

	if w.FlushInterval > 0 {
		go func() {
			for {
				time.Sleep(time.Second * time.Duration(w.FlushInterval))
				forceFlush <- true

				// Send a flush command to unblock the fifo read in case no
				// lines are being written to the fifo. This command is
				// ignored below, the forceFlush channel is what matters.
				w.Fifo.SendCommand("flush")
			}
		}()
	}

	chunk, key, flush := w.reset()
	for line := range lines {

		select {
		case <-forceFlush:
			logger.Debug("force flush signal received")
			flush = true
		default:
			flush = false
		}

		// The flush command was sent to unblock the fifo read in case no
		// lines were being written to the fifo. The forceFlush channel is
		// what matters, so we ignore the flush command and move on.
		if line != ".flush" {
			chunk[key] = line
			key++
		} else {
			logger.Debug("command received: flush")
		}

		if key >= w.QueueLimit {
			flush = true
		}

		if flush {
			logger.Debug("flush buffer: %v items in queue", key)
			chunks <- chunk[:key]
			chunk, key, flush = w.reset()
		}
	}

	// We stopped reading the fifo, so flush anything left in the buffer.
	if !flush {
		logger.Debug("flush buffer: %v items in queue", key)
		chunks <- chunk[:key]
	}
}

// LoggerBufferFlusher implements BufferFlusher and is useful for debugging
// and development of the fifo2kinesis app. It processes lines by streaming
// them as INFO level log messages.
type LoggerBufferFlusher struct{}

// Flush streams the data set to it from the BufferWriter as INFO level log
// messages. If never writes anything to the failed channel.
//
// TODO Would it be useful to be able to send a certain percentage of lines
// to the failed channel for testing the retry capabilities?
func (f *LoggerBufferFlusher) Flush(chunks <-chan []string, failed chan []string) {
	for chunk := range chunks {
		for _, line := range chunk {
			logger.Info(line)
		}
	}
}

// NullFailedAttemptHandler implements FailedAttemptHandler and basically
// drops all failed attempts.
type NullFailedAttemptHandler struct{}

// SaveAttempt does nothing with the data passed to it.
func (h NullFailedAttemptHandler) SaveAttempt(attempt []string) error {
	return nil
}

// Retry does nothing, since no attemtps are ever saved by SaveAttempt.
func (h NullFailedAttemptHandler) Retry() {}
