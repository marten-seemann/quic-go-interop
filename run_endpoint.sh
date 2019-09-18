#!/bin/bash
set -e

# Set up the routing needed for the simulation.
/setup.sh

if [ "$ROLE" == "client" ]; then
    # Wait for the simulator to start up.
    /wait-for-it.sh sim:57832 -s -t 5
    echo "Starting QUIC client..."
    echo "Client params: $CLIENT_PARAMS"
    echo "Test case: $TESTCASE"
    go run client/main.go $CLIENT_PARAMS
else
    echo "Running QUIC server on 0.0.0.0:4433"
    echo "$@"
    go run server/main.go "$@"
fi
