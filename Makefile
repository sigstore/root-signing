SHELL := /bin/bash
OUTPUT_FORMAT = $(shell if [ "${GITHUB_ACTIONS}" == "true" ]; then echo "github"; else echo ""; fi)
GOBIN ?= $(shell go env GOPATH)/bin

.PHONY: help
help: ## Shows all targets and help from the Makefile (this message).
	@echo "root-signing Makefile"
	@echo "Usage: make [COMMAND]"
	@echo ""
	@grep --no-filename -E '^([/a-z.A-Z0-9_%-]+:.*?|)##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = "(:.*?|)## ?"}; { \
			if (length($$1) > 0) { \
				printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2; \
			} else { \
				if (length($$2) > 0) { \
					printf "%s\n", $$2; \
				} \
			} \
		}'

## Linters
#####################################################################

lint: ## Run all linters.
lint: golangci-lint yamllint

golangci-lint: ## Runs the golangci-lint linter.
	@set -e;\
		extraargs=""; \
		if [ "$(OUTPUT_FORMAT)" == "github" ]; then \
			extraargs="--out-format github-actions"; \
		fi; \
		$(GOBIN)/golangci-lint run -c .golangci.yml ./... $$extraargs

yamllint: ## Runs the yamllint linter.
	@set -e;\
		extraargs=""; \
		if [ "$(OUTPUT_FORMAT)" == "github" ]; then \
			extraargs="-f github"; \
		fi; \
		yamllint -c .yamllint.yaml . $$extraargs

.PHONY: keygen
keygen:
	@go build -tags=pivkey -o $@ ./tests/keygen

.PHONY: tuf
tuf:
	@go build -tags=pivkey -o $@ ./cmd/tuf

.PHONY: test
test:
	@go test -tags=pivkey ./...
