#!/bin/sh

cd $(dirname $0)/..
GOPATH=$PWD

version=$(git describe --exact-match --tags HEAD 2> /dev/null)
if [ -z "$version" ]; then
  version=$(git rev-parse --short HEAD)
fi

GOOS=linux GOARCH=amd64 gb build
GOOS=darwin GOARCH=amd64 gb build

(
  cd bin
  mv fifo2kinesis-linux-amd64 fifo2kinesis-$version-linux-amd64
  mv fifo2kinesis-darwin-amd64 fifo2kinesis-$version-darwin-amd64
)

