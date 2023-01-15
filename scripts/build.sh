#!/usr/bin/env bash

set -e

rm -f bin/*
mkdir -p bin/

go build -v ./cmd/dmud