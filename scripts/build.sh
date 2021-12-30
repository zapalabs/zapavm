#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Load the constants
# Set the PATHS
GOPATH="$(go env GOPATH)"

# zcashvm root directory
zcashvm_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

# Set default binary directory location
binary_directory="$GOPATH/src/github.com/ava-labs/avalanchego/build/plugins"
name="tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH"

if [[ $# -eq 1 ]]; then
    binary_directory=$1
elif [[ $# -eq 2 ]]; then
    binary_directory=$1
    name=$2
elif [[ $# -ne 0 ]]; then
    echo "Invalid arguments to build zcashvm. Requires either no arguments (default) or one arguments to specify binary location."
    exit 1
fi


# Build zcashvm, which is run as a subprocess
echo "Building zcashvm in $binary_directory/$name"
go build -o "$binary_directory/$name" "main/"*.go
