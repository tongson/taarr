.ONESHELL:
.SILENT:
.SHELL := /usr/bin/env bash
.PHONY: staticcheck errcheck fmt build test clean
SRC= "cmd/rr/... internal/lib/..."
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
	cd tools
	GO111MODULE=on go build -o ../bin/golint golang.org/x/lint/golint
	GO111MODULE=on go build -o ../bin/staticcheck honnef.co/go/tools/cmd/staticcheck
	GO111MODULE=on go build -o ../bin/errcheck github.com/kisielk/errcheck

fmt:
	@echo "$(BLUE)$(TIME)$(GREEN) + go fmt $(RESET)"
	@go fmt cmd/rr/main.go
	@go fmt internal/lib/lib.go

errcheck:
	@echo "$(BLUE)$(TIME)$(GREEN) + errcheck $(RESET)"
	bin/errcheck "$(SRC)"

staticcheck:
	@echo "$(BLUE)$(TIME)$(GREEN) + staticheck $(RESET)"
	bin/staticcheck "$(SRC)" 

lint:
	@echo "$(BLUE)$(TIME)$(GREEN) + golint $(RESET)"
	bin/golint "cmd/rr/main.go"

check: errcheck staticcheck lint
	@echo "$(BLUE)$(TIME)$(GREEN) + CHECK DONE$(RESET)"

build: fmt
	@go mod tidy
	@echo "$(BLUE)$(TIME)$(GREEN) + BUILD START$(RESET)"
	@mkdir -p bin
	#@/usr/bin/env GOOS=linux go build -o bin/rr -ldflags="-s -w" ./...
	@/usr/bin/env GOARCH=386 GOOS=linux CGO_ENABLED=0 go build -trimpath -o bin/rr -ldflags '-s -w' ./cmd/rr
	@echo "$(BLUE)$(TIME)$(CYAN) ! BUILD DONE $(RESET)"

test:
	@echo "$(BLUE)$(TIME)$(YELLOW) + TEST START $(RESET)"
	cd .test 
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 1$(RESET)"
	../bin/rr local test:args1 --one --two --three
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 2$(RESET)"
	../bin/rr local test:args2 one 1
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 3$(RESET)"
	../bin/rr local test:args3 -v
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 4$(RESET)"
	../bin/rr local test/args4/1
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 5$(RESET)"
	../bin/rr test/args4/1
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 6$(RESET)"
	../bin/rr test/args6/1 2 3 4
	@echo "$(BLUE)$(TIME)$(MAGENTA) . untar files $(RESET)"
	../bin/rr local test:files
	@echo "$(BLUE)$(TIME)$(MAGENTA) . failure conditioin $(RESET)"
	../bin/rr local test:fail || true
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 1$(RESET)"
	../bin/rr local test/args1 --one --two --three
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 2$(RESET)"
	../bin/rr local test/args2 one 1
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 3$(RESET)"
	../bin/rr local test/args3 -v
	@echo "$(BLUE)$(TIME)$(MAGENTA) . untar files $(RESET)"
	../bin/rr local test/files
	@echo "$(BLUE)$(TIME)$(MAGENTA) . failure conditioin $(RESET)"
	../bin/rr local test:fail || true
	rmdir /YYY
	rm -f /XXX
	@echo "$(BLUE)$(TIME)$(CYAN) ! TEST DONE $(RESET)"

clean:
	rm -f bin/rr

