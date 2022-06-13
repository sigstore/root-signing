#!/bin/bash

# Print all commands and stop on errors
set -ex

source "./scripts/utils.sh"

# Check that a github user is set.
check_user

# Set REPO
set_repository

# Dump the git state and clean-up
print_git_state
clean_state

# Checkout the working branch
checkout_branch

# Ask user to insert key 
read -n1 -r -s -p "Insert your Yubikey, then press any key to continue...\n" 

# Add the key!
./tuf add-key -repository $REPO

# Ask user to remove key (and replace with SSH security key)
read -n1 -r -s -p "Remove your Yubikey, then press any key to continue...\n" 

commit_and_push_changes add-key
