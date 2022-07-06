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

# Sign the root and targets
./tuf publish -repository $REPO
# Clear and copy into the repository/
rm -r repository/
cp -r $REPO/ repository/

commit_and_push_changes publish