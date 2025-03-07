#!/bin/bash

# Set environment variables
export SERVER_IP="192.168.1.10"
export SERVER_PORT="8080"

# Build the project (optional, if you haven't built it yet)
go build -o game main.go

# Run the application
./game