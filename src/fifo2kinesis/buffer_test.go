package main

import (
	"os"
	"testing"
	"time"
)

func TestBufferFlushLimitExceeded(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	bw := &MemoryBufferWriter{
		Fifo:          fifo,
		FlushInterval: 0,
		QueueLimit:    2,
	}

	lines := make(chan string)
	chunks := make(chan []string)

	go func() {
		bw.Write(lines, chunks)
	}()

	go func() {
		lines <- "zero"
		lines <- "one"
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
		if got[0] != "zero" || got[1] != "one" {
			t.Errorf("buffer flush limit exceeded test failed: got %q", got)
		}
	}
}

func TestBufferFlushWithinLimit(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	bw := &MemoryBufferWriter{
		Fifo:          fifo,
		FlushInterval: 0,
		QueueLimit:    2,
	}

	lines := make(chan string)
	chunks := make(chan []string)

	go func() {
		bw.Write(lines, chunks)
	}()

	go func() {
		lines <- "zero"
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

func TestBufferFlushInterval(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	bw := &MemoryBufferWriter{
		Fifo:          fifo,
		FlushInterval: 1,
		QueueLimit:    2,
	}

	lines := make(chan string)
	chunks := make(chan []string)

	go func() {
		fifo.Scan(lines)
	}()

	go func() {
		bw.Write(lines, chunks)
	}()

	go func() {
		fifo.Writeln("zero")
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
		if got[0] != "zero" {
			t.Errorf("buffer flush interval test failed: got %q", got)
		}
	}
}
