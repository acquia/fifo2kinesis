package main

import (
	"bufio"
	"os"
)

// Fifo represents the named pipe.
type Fifo struct {
	Name string
}

// Writes a line to the fifo.
func (f *Fifo) Writeln(s string) error {
	return f.WriteString(s + "\n")
}

// Write writes a string to the fifo.
func (f *Fifo) WriteString(s string) error {
	file, err := os.OpenFile(f.Name, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.WriteString(s)

	return err
}

// SendCommand writes a line to the fifo that the pipeline interprets as
// a command to perform some action. This is necessary because reading from
// the fifo blocks until something is written to it. This is how we chose to
// get around this challenge.
func (f *Fifo) SendCommand(cmd string) (err error) {
	err = f.Writeln("." + cmd)
	logger.Debug("command sent: %s", cmd)
	return
}

// Scan reads lines from the fifo and sends them to the out channel. The
// only ways to stop the scan is to write the ".stop" string to the fifo
// or if there is an error reading data from the fifo.
func (f *Fifo) Scan(out chan string) error {
	stop := false
	for {

		// This statement blocks until a line is written to the fifo. This
		// is why we have the write ".stop" to the fifo during shutdown.
		file, err := os.OpenFile(f.Name, os.O_RDONLY, os.ModeNamedPipe)
		if err != nil {
			return err
		}

		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		// If we break the loop we lose data. Not sure how to handle that.
		// Might have to live with the loop running through until the end.
		// Of course, the app writing to the fifo could continue writing
		// lines which would prevent fifo2kinesis from shutting down, but
		// that seems to be the best of the worst options.
		//
		// If you are reading this, consider yourself challenged to find a
		// better way of doing this. Most people are smarter than me, so I
		// bet you can do it. I'm rooting for you!
		//
		for scanner.Scan() {
			line := scanner.Text()
			switch line {
			case ".stop":
				logger.Debug("command received: stop")
				stop = true
			default:
				out <- line
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
