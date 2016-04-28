package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// FileFailedAttemptHandler implements FailedAttemptHandler and captures
// failed attempts in files for retry at a later time.
//
// dir is the directory where files are written.
//
// fifo is a Fifo that models the named pipe being read. It is also used to
// write failed attempts back to the named pipe.
type FileFailedAttemptHandler struct {
	dir  string
	fifo *Fifo
}

// Filepath returns the full path to a new retry file.
func (h *FileFailedAttemptHandler) Filepath() string {
	date := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("%s/fifo2kinesis-%s-%s", h.dir, date, RandomString(8))
}

// SaveAttempt saves failed attempts to a file for retry at a later time via
// the Retry method.
func (h *FileFailedAttemptHandler) SaveAttempt(attempt []string) error {

	// TODO Add duplicate file detection when creating retry files
	// https://github.com/acquia/fifo2kinesis/issues/21
	file, err := os.OpenFile(h.Filepath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.WriteString(strings.Join(attempt, "\n"))

	return err
}

// Files returns all retry files in the
func (h *FileFailedAttemptHandler) Files() []string {

	files, err := ioutil.ReadDir(h.dir)
	if err != nil {
		return []string{}
	}

	filepaths := make([]string, len(files))
	for key, file := range files {
		filepaths[key] = h.dir + "/" + file.Name()
	}

	return filepaths
}

// Retry processes a group of files and writes the lines back to the FIFO so
// that they go through the pipeline again.
func (h *FileFailedAttemptHandler) Retry() {
	// TODO Make the max number of retry attempts configurable
	// https://github.com/acquia/fifo2kinesis/issues/20
	i := 0

	for _, filepath := range h.Files() {

		h.RetryAttempt(filepath)

		i++
		if i >= 3 {
			return
		}
	}
}

// RetryAttempt reads the lines from filename and writes them back to the
// FIFO so that they go through the pipeline again.
func (h *FileFailedAttemptHandler) RetryAttempt(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// All TODO's related to https://github.com/acquia/fifo2kinesis/issues/22

	// TODO capture lines that failed and write a new file?
	for scanner.Scan() {
		line := scanner.Text()
		h.fifo.Writeln(line)
	}

	// TODO handle scanner errors?
	//	if err := scanner.Err(); err != nil {
	//		return err
	//	}

	// TODO handle file removal errors?
	os.Remove(filename)

	return nil
}
