.ONESHELL:
.SILENT:
.SHELL := /usr/bin/env bash
.PHONY: errcheck vet fmt build test clean
BOLD=$(shell tput bold)
RED=$(shell tput setaf 1)
GREEN=$(shell tput setaf 2)
YELLOW=$(shell tput setaf 3)
BLUE=$(shell tput setaf 4)
MAGENTA=$(shell tput setaf 5)
CYAN=$(shell tput setaf 6)
RESET=$(shell tput sgr0)
TIME=$(shell date "+%Y-%m-%d %H:%M:%S")
all: build test

setup:
	@go get -u "github.com/kisielk/errcheck"
	@zypper --non-interactive install --no-recommends install upx

errcheck:
	@echo "$(BLUE)$(TIME)$(GREEN) + errcheck $(RESET)"
	@~/go/bin/errcheck ./...

vet:
	@echo "$(BLUE)$(TIME)$(GREEN) + go vet $(RESET)"
	@go vet ./...

fmt:
	@echo "$(BLUE)$(TIME)$(GREEN) + go fmt $(RESET)"
	@go fmt cmd/rr/main.go
	@go fmt pkg/aux/aux.go

build: fmt vet errcheck
	@go mod tidy
	@echo "$(BLUE)$(TIME)$(GREEN) + BUILD START$(RESET)"
	@mkdir -p bin/
	#@go build ./cmd/rr 
	@/usr/bin/env GOOS=linux go build -ldflags="-s -w" ./...
	@mv rr bin/rr
	@echo "$(BLUE)$(TIME)$(CYAN) ! BUILD DONE $(RESET)"

release: build
	@echo "$(BLUE)$(TIME)$(GREEN) + COMPRESS START$(RESET)"
	@upx --brute bin/rr
	@echo "$(BLUE)$(TIME)$(CYAN) ! COMPRESS DONE $(RESET)"

test:
	@echo "$(BLUE)$(TIME)$(YELLOW) + TEST START $(RESET)"
	cd .test 
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 1$(RESET)"
	../bin/rr local test:args1 --one --two --three
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 2$(RESET)"
	../bin/rr local test:args2 one 1
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 3$(RESET)"
	../bin/rr local test:args3 -v
	@echo "$(BLUE)$(TIME)$(MAGENTA) . untar files $(RESET)"
	../bin/rr local test:_files
	@echo "$(BLUE)$(TIME)$(MAGENTA) . failure conditioin $(RESET)"
	../bin/rr local test:fail || true
	@echo "$(BLUE)$(TIME)$(CYAN) ! TEST DONE $(RESET)"

clean:
	rm -f bin/rr

