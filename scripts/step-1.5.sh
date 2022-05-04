#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi
if [ -z "$TIMESTAMP_KEY" ]; then
    echo "Set TIMESTAMP_KEY"
    exit
fi
if [ -z "$SNAPSHOT_KEY" ]; then
    echo "Set SNAPSHOT_KEY"
    exit
fi
if [ -z "$REKOR_KEY" ]; then
    echo "Set REKOR_KEY"
    exit
fi
if [ -z "$STAGING_KEY" ]; then
    echo "Set STAGING_KEY"
    exit
fi
# Repo options
if [ -z "${PREV_REPO+set}" ]; then
    echo "Set PREV_REPO"
    exit
fi
if [ -z "$REPO" ]; then
    REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')
    echo "Using default REPO $REPO"
fi

if [ -z "$BRANCH" ]; then
    export BRANCH=main
else
    echo "Using branch $BRANCH"
fi

# Dump the git state
git status
git remote -v

if [ -n "$NO_CLEAN" ]; then
    git clean -d -f
    git checkout $BRANCH
    git pull upstream $BRANCH
fi

git status

# Copy the previous keys and repository into the new repository.
if [ ! -z "$PREV_REPO" ]; then
    cp -pr ${PREV_REPO}/* ${REPO}
fi
mkdir -p ${REPO}/staged/targets

# Remove a key by ID that need to be removed from the root keyholders
if [[ -n $1 ]]; then 
    echo "Removing key: $1"
    rm -r ${REPO}/keys/$1
fi

# TODO: Remove when v3-staging is merged
if [[ $BRANCH == "v3-staging" ]]; then
    if [ -z "$REVOCATION_KEY" ]; then
        echo "Set REVOCATION_KEY"
        exit
    fi
    # Setup the root and targets
    ./tuf init -repository $REPO -target-meta config/targets-metadata.yml -snapshot ${SNAPSHOT_KEY} -timestamp ${TIMESTAMP_KEY} -previous "${PREV_REPO}"
    # Add rekor delegation
    cp targets/rekor.pub targets/rekor.0.pub
    ./tuf add-delegation -repository $REPO -name "rekor" -key $REKOR_KEY -path "rekor.*.pub" -target-meta config/rekor-metadata.yml
    # Add staging project delegation
    ./tuf add-delegation -repository $REPO -name "staging" -key $STAGING_KEY -path "*"
    # Add revoked project delegation
    ./tuf add-delegation -repository $REPO -name "revocation" -key $REVOCATION_KEY -path "*" -target-meta config/revocation-metadata.yml
else
    # Setup the root and targets
    ./tuf init -repository $REPO -target targets/fulcio.crt.pem -target targets/fulcio_v1.crt.pem -target targets/rekor.pub -target targets/ctfe.pub -target targets/artifact.pub -snapshot ${SNAPSHOT_KEY} -timestamp ${TIMESTAMP_KEY} -previous ${PREV_REPO}
    # Add rekor delegation
    cp targets/rekor.pub targets/rekor.0.pub
    ./tuf add-delegation -repository $REPO -name "rekor" -key $REKOR_KEY -path "rekor.*.pub" -target targets/rekor.0.pub
    # Add staging project delegation
    ./tuf add-delegation -repository $REPO -name "staging" -key $STAGING_KEY -path "*"
fi

if [ -n "$NO_PUSH" ]; then
    echo "Skipping push, exiting early..."
    exit
fi

git checkout -b setup-root
git add ceremony/
git commit -s -a -m "Setting up root for ${GITHUB_USER}"
git push -f origin setup-root

# Open the browser
export GITHUB_URL=$(git remote -v | awk '/^upstream/{print $2}'| head -1 | sed -Ee 's#(git@|git://)#https://#' -e 's@com:@com/@' -e 's#\.git$##')
export CHANGE_BRANCH=$(git symbolic-ref HEAD | cut -d"/" -f 3,4)
export PR_URL=${GITHUB_URL}"/compare/${BRANCH}..."${CHANGE_BRANCH}"?expand=1"
open "${PR_URL}" || xdg-open "${PR_URL}"
