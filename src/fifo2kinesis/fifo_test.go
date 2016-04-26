package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TempFifo(t *testing.T) *Fifo {

	name := os.TempDir() + "/fifo2kinesis-" + RandomString(8) + ".pipe"
	err := syscall.Mkfifo(name, 0600)
	if err != nil {
		t.Errorf("error creating fifo: %s", err)
	}

	return &Fifo{name}
}

func TestFifoWriteAndScan(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	out := make(chan string, 1)

	go func() {
		fifo.Scan(out)
	}()

	go func() {
		if err := fifo.Writeln("test"); err != nil {
			t.Errorf("fifo write test failed: %s", err)
		}
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * 3)
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
	fifo := TempFifo(t)
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

func TestScanDrain(t *testing.T) {
	fifo := TempFifo(t)
	defer os.Remove(fifo.Name)

	out := make(chan string, 1)

	go func() {
		fifo.Scan(out)
		close(out)
	}()

	go func() {
		// Write four lines inclusing a ".stop" comman in the middle. If all
		// goes well we should read three lines from the out channel.
		if err := fifo.WriteString("zero\n.stop\none\ntwo"); err != nil {
			t.Errorf("error sending stop command: %s", err)
		}
	}()

	done := make(chan bool)

	go func() {
		key := 0
		lines := make([]string, 3)
		for line := range out {
			lines[key] = line
			key++
		}

		if key != 3 {
			t.Errorf("fifo scan drain test failed: got %q lines", key)
		}

		if lines[0] != "zero" || lines[1] != "one" || lines[2] != "two" {
			t.Errorf("fifo scan drain test failed: got %q", lines)
		}

		done <- true
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * 3)
		timeout <- true
	}()

	select {
	case <-timeout:
		t.Error("timeout waiting scan drain test to complete")
	case <-done:
	}
}
