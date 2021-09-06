.ONESHELL:
.SILENT:
.SHELL := /usr/bin/env bash
.PHONY: test
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

test:
	@echo "$(BLUE)$(TIME)$(YELLOW) + TEST START $(RESET)"
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 1$(RESET)"
	../bin/rr local test:args1 --one --two --three
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 2$(RESET)"
	../bin/rr local test:args2 one 1
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 3$(RESET)"
	../bin/rr local test:args3 -v
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 4$(RESET)"
	../bin/rr local test:args4:1
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 5$(RESET)"
	../bin/rr test:args4:1
	@echo "$(BLUE)$(TIME)$(MAGENTA) . arguments handling 6$(RESET)"
	../bin/rr test:args6:1 2 3 4
	@echo "$(BLUE)$(TIME)$(MAGENTA) . untar files $(RESET)"
	../bin/rr local test:files
	@echo "$(BLUE)$(TIME)$(MAGENTA) . failure conditioin $(RESET)"
	../bin/rr local test:fail || true
	@echo "$(BLUE)$(TIME)$(CYAN) ! TEST DONE $(RESET)"
