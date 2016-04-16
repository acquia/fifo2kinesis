package main

import (
	"bufio"
	"os"
)

// Fifo represents the named pipe.
type Fifo struct {
	Name string
}

// SendCommand writes a line to the fifo that the pipeline interprets as
// a command to perform some action.
func (f *Fifo) SendCommand(cmd string) error {
	file, err := os.OpenFile(f.Name, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.WriteString("." + cmd)
	logger.Debug("command sent: %s", cmd)

	return err
}

// Scan reads lines from the fifo and sends them to the out channel. The
// only ways to stop the scan is to write the ".stop" string to the fifo
// or if there is an error reading data from the fifo.
func (f *Fifo) Scan(out chan string) error {
	stop := false
	for {

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
		// lines which would prevent the app from shutting down, but that
		// seems to be the best of the worst options.
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
