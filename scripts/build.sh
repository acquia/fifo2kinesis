#!/bin/sh

cd $(dirname $0)/..
GOPATH=$PWD

go build -o bin/fifo2kinesis fifo2kinesis

