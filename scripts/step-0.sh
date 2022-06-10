#!/bin/bash

# Print all commands and stop on errors
set -ex

source "./scripts/utils.sh"

# Check that a github user is set.
check_user

# Dump the git state
print_git_state

# Setup forks
setup_forks

# Cleanup branches
cleanup_branchs
clean_state

# Checkout the working branch
checkout_branch

# build the tuf binary
go build -o tuf -tags=pivkey ./cmd/tuf
