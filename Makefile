.ONESHELL:
.SILENT:
.SHELL := /usr/bin/env bash
.PHONY: errcheck fmt build clean check lint
SRC= "main.go"
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

fmt:
	@echo "$(BLUE)$(TIME)$(GREEN) + go fmt $(RESET)"
	@go fmt main.go

errcheck:
	@echo "$(BLUE)$(TIME)$(GREEN) + errcheck $(RESET)"
	errcheck "$(SRC)"

lint:
	@echo "$(BLUE)$(TIME)$(GREEN) + golint $(RESET)"
	golint "main.go"

check: errcheck lint
	@echo "$(BLUE)$(TIME)$(GREEN) + CHECK DONE$(RESET)"

build:
	@echo "$(BLUE)$(TIME)$(GREEN) + BUILD START$(RESET)"
	@mkdir -p bin
	@/usr/bin/env CGO_ENABLED=0 go build -trimpath -o bin/rr -ldflags '-s -w'
	@echo "$(BLUE)$(TIME)$(CYAN) ! BUILD DONE $(RESET)"

clean:
	rm -f bin/rr

