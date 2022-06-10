#!/bin/bash

check_user() {
    if [ -z "$GITHUB_USER" ]; then
        echo "Set GITHUB_USER"
        exit
    fi
}

check_initialization_options() {
    exit
}

set_repository() {
    if [ -z "$REPO" ]; then
        REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')
    fi
    echo "Using REPO $REPO"
}

print_git_state() {
    git status
    git remote -v
}

checkout_branch() {
    if [ -z "$BRANCH" ]; then
        BRANCH=main
    fi
    echo "Working from branch $BRANCH"
    git checkout $BRANCH
    if [ -n "$LOCAL" ]; then
        echo "Working on local changes. There may be uncommitted changes, so skipping upstream pull..."
    else 
        git pull upstream $BRANCH
    fi
    git rev-parse HEAD
}

cleanup_branchs() {
    git branch -D setup-root || true
    git branch -D add-key || true
    git branch -D sign-root-targets || true
    git branch -D sign-delegations || true
    git branch -D snapshot-timestamp || true
    git branch -D publish || true
}

setup_forks() {
    git remote rm upstream || true
    git remote add upstream git@github.com:sigstore/root-signing.git
    git remote rm origin || true
    git remote add origin git@github.com:"$GITHUB_USER"/root-signing.git
    git remote -v
}

# clean_state cleans up the current git state, unless $LOCAL_TESTING is set.
clean_state() {
    if [ -n "$LOCAL" ]; then
        echo "Skipping clean, returning early..."
        return
    fi
    git clean -d -f
}

# commit_and_push_changes requires an argument to push changes to, unless LOCAL_TESTING is set.
commit_and_push_changes() {
    git status

    if [ -n "$LOCAL" ]; then
        echo "Skipping push, exiting early..."
        return
    fi

    if [ -z "$1" ]; then
        echo "Commit branch missing..."
        exit
    fi

    # Create a commit
    git checkout -b $1
    git add ceremony/ repository/
    git commit -s -a -m "$1 for ${GITHUB_USER}"
    git push -f origin $1

    # Open the browser
    export GITHUB_URL=$(git remote -v | awk '/^upstream/{print $2}'| head -1 | sed -Ee 's#(git@|git://)#https://#' -e 's@com:@com/@' -e 's#\.git$##')
    export CHANGE_BRANCH=$(git symbolic-ref HEAD | cut -d"/" -f 3,4)
    export PR_URL=${GITHUB_URL}"/compare/${BRANCH}..."${GITHUB_USER}:${CHANGE_BRANCH}"?expand=1"
    open "${PR_URL}" || xdg-open "${PR_URL}"
}
