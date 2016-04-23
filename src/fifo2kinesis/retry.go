package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type FileFailedAttemptHandler struct {
	dir  string
	fifo *Fifo
}

func (h *FileFailedAttemptHandler) Filepath() string {
	date := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("%s/fifo2kinesis-%s-%s", h.dir, date, RandomString(8))
}

func (h *FileFailedAttemptHandler) SaveAttempt(attempt []string) error {

	// TODO although a colision is unlikely, we should loop until we find a
	// unique filename like TempFile()
	file, err := os.OpenFile(h.Filepath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.WriteString(strings.Join(attempt, "\n"))

	return err
}

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

func (h *FileFailedAttemptHandler) Retry() {
	// TODO make the max number of attempts configurable.
	i := 0

	for _, filepath := range h.Files() {

		h.RetryAttempt(filepath)

		i++
		if i >= 3 {
			return
		}
	}
}

func (h *FileFailedAttemptHandler) RetryAttempt(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// TODO capture lines that failed and write a new file.
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
