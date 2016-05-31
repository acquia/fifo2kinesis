package main

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// TestBufferFlushLimitExceeded tests that the buffer is flushed when then
// number of ites reaches the queue limit.
func TestBufferFlushLimitExceeded(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	bw := &MemoryBufferWriter{
		Fifo:          fifo,
		FlushInterval: 0,
		QueueLimit:    2,
	}

	lines := make(chan []byte)
	chunks := make(chan [][]byte)
	zero := []byte("zero")
	one := []byte("one")

	go func() {
		bw.Write(lines, chunks)
	}()

	go func() {
		lines <- zero
		lines <- one
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * 3)
		timeout <- true
	}()

	select {
	case <-timeout:
		t.Error("timeout waiting for buffer flush limit exceeded test to complete")
	case got := <-chunks:
		if !bytes.Equal(got[0], zero) || !bytes.Equal(got[1], one) {
			t.Errorf("buffer flush limit exceeded test failed: got %q", got)
		}
	}
}

// TestBufferFlushWithinLimit tests that the buffer is NOT flushed if the
// number of items is less than the queue limit and there is no flush
// interval set. The expected behavior is for the the timeout condition to
// be reached.
func TestBufferFlushWithinLimit(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	bw := &MemoryBufferWriter{
		Fifo:          fifo,
		FlushInterval: 0,
		QueueLimit:    2,
	}

	lines := make(chan []byte)
	chunks := make(chan [][]byte)
	zero := []byte("zero")

	go func() {
		bw.Write(lines, chunks)
	}()

	go func() {
		lines <- zero
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * 3)
		timeout <- true
	}()

	select {
	case <-timeout:
	case <-chunks:
		t.Error("expected buffer flush within limit test to timeout")
	}
}

// TestBufferFlushInterval tests that the bufer is flushed within the
// interval specified, even if the number of items is less than the queue
// limit.
func TestBufferFlushInterval(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	bw := &MemoryBufferWriter{
		Fifo:          fifo,
		FlushInterval: 1,
		QueueLimit:    2,
	}

	lines := make(chan []byte)
	chunks := make(chan [][]byte)
	zero := []byte("zero")

	go func() {
		fifo.Scan(lines)
	}()

	go func() {
		bw.Write(lines, chunks)
	}()

	go func() {
		fifo.Writeln(zero)
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * 3)
		timeout <- true
	}()

	select {
	case <-timeout:
		t.Error("timeout waiting for buffer flush interval test to complete")
	case got := <-chunks:
		if !bytes.Equal(got[0], zero) {
			t.Errorf("buffer flush interval test failed: expected timeout, got %q", got)
		}
	}
}

// TestBufferChunkSize
func TestBufferChunkSize(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	bw := &MemoryBufferWriter{
		Fifo:          fifo,
		FlushInterval: 1,
		QueueLimit:    2,
	}

	lines := make(chan []byte)
	chunks := make(chan [][]byte)
	zero := []byte("zero")

	go func() {
		fifo.Scan(lines)
	}()

	go func() {
		bw.Write(lines, chunks)
	}()

	go func() {
		fifo.Writeln(zero)
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * 3)
		timeout <- true
	}()

	select {
	case <-timeout:
		t.Error("timeout waiting for buffer chunk size test to complete")
	case got := <-chunks:
		if len(got) != 1 {
			t.Errorf("buffer flush interval test failed: expected length of 1, got %v", len(got))
		}
	}

}
