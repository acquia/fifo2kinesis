package main

import (
	"bufio"
	"bytes"
	"os"
)

// Fifo represents the named pipe. It contains methods that write to and
// continuously read from the named pipe.
//
// Name is the absolute path to the named pipe.
type Fifo struct {
	Name string
}

// Writeln writes a line to the FIFO, suffixed with a Unix new line.
func (f *Fifo) Writeln(b []byte) error {
	b = append(b, byte(10))
	return f.Write(b)
}

// WriteString writes a string as-is to the fifo. The Writeln method is
// usually used in favor of this one.
func (f *Fifo) Write(b []byte) error {
	file, err := os.OpenFile(f.Name, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.Write(b)

	return err
}

// SendCommand writes a line to the fifo that the pipeline interprets as
// some action to perform. This is necessary because reading from the FIFO
// blocks until something is written to it.
//
// TODO This is how I chose to solve this problem. It is admittedly a hack.
// There are some pretty smart people out there, I am hoping that someone
// smarter than me can find a better way of doing things and eliminate the
// need for this method. If you are reading this, consider yourself to be
// officially presented with this challenge.
func (f *Fifo) SendCommand(cmd string) (err error) {
	err = f.Writeln([]byte("." + cmd))
	logger.Debug("command sent: %s", cmd)
	return
}

// Scan reads lines from the fifo and sends them to the out channel. The
// only ways to stop the scan is to write the ".stop" string to the fifo
// or if there is an error reading data from the fifo.
func (f *Fifo) Scan(out chan []byte) error {
	stop := false

	// This statement blocks until a line is written to the fifo. This
	// is why we have the write ".stop" to the fifo during shutdown.
	file, err := os.OpenFile(f.Name, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return err
	}

	defer file.Close()

	// If we break the loop we lose data. Not sure how to handle that.
	// Might have to live with the loop running through until the end.
	// Of course, the app writing to the fifo could continue writing
	// lines which would prevent fifo2kinesis from shutting down, but
	// that seems to be the best of the worst options.
	//
	// You. Yes, you. Please find a better solution.
	stop_cmd := []byte(".stop")
	for {
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := scanner.Bytes()
			if bytes.Equal(line, stop_cmd) {
				logger.Debug("command received: stop")
				stop = true
			} else {
				bytes := make([]byte, len(line))
				copy(bytes, line)
				out <- bytes
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}
		if stop {
			return nil
		}
	}

	return nil
}
