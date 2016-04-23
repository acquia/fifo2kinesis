package main

import "time"

type BufferWriter interface {
	Write(lines <-chan string, chunks chan []string)
}

type BufferFlusher interface {
	Flush(chunks <-chan []string, failed chan []string)
}

type FailedAttemptHandler interface {
	SaveAttempt(attempt []string) error
	Retry()
}

type Buffer struct {
	BufferWriter
	BufferFlusher
	FailedAttemptHandler
}

type MemoryBufferWriter struct {
	Fifo          *Fifo
	FlushInterval int
	QueueLimit    int
}

func (w *MemoryBufferWriter) reset() ([]string, int, bool) {
	return make([]string, w.QueueLimit), 0, false
}

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

type LoggerBufferFlusher struct{}

func (f *LoggerBufferFlusher) Flush(chunks <-chan []string, failed chan []string) {
	for chunk := range chunks {
		for _, line := range chunk {
			logger.Info(line)
		}
	}
}

type NullFailedAttemptHandler struct{}

func (h NullFailedAttemptHandler) SaveAttempt(attempt []string) error {
	return nil
}

func (h NullFailedAttemptHandler) Retry() {}
