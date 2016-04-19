package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func NewFifo(t *testing.T) *Fifo {

	name := os.TempDir() + "/fifo2kinesis-" + RandomString(8) + ".pipe"
	err := syscall.Mkfifo(name, 0600)
	if err != nil {
		t.Errorf("error creating fifo: %s", err)
	}

	return &Fifo{name}
}

func TestFifoWriteScan(t *testing.T) {
	fifo := NewFifo(t)
	defer os.Remove(fifo.Name)

	out := make(chan string, 1)

	go func() {
		fifo.Scan(out)
	}()

	go func() {
		if err := fifo.WriteString("test"); err != nil {
			t.Errorf("fifo write test failed: %s", err)
		}
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * 5)
		timeout <- true
	}()

	select {
	case <-timeout:
		t.Error("timeout waiting for line to be read from fifo")
	case line := <-out:
		if line != "test" {
			t.Errorf("fifo scan test failed: got %q", line)
		}
	}
}

func TestStopCommand(t *testing.T) {
	fifo := NewFifo(t)
	defer os.Remove(fifo.Name)

	out := make(chan string, 1)
	stopped := make(chan bool, 1)

	go func() {
		fifo.Scan(out)
		stopped <- true
	}()

	go func() {
		if err := fifo.SendCommand("stop"); err != nil {
			t.Errorf("error sending stop command: %s", err)
		}
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * 3)
		timeout <- true
	}()

	select {
	case <-timeout:
		t.Error("timeout waiting scanning to stop")
	case <-stopped:
	}
}
