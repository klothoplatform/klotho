#!/bin/bash

# Get the directory of the current script
SCRIPT_DIR=$(dirname "$0")

# Navigate to the script directory
cd "$SCRIPT_DIR"

# For Go
protoc --go_out=./go --go_opt=paths=source_relative --proto_path=. service.proto
protoc --go-grpc_out=./go --go-grpc_opt=paths=source_relative --proto_path=. service.proto

# For Python
pipenv run python3 -m grpc_tools.protoc -I. --python_out=./python/ --grpc_python_out=./python/ service.proto
pipenv run python3 ./python/generate_and_adjust_grpc.py