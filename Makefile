ROOT_DIR = $(CURDIR)
OUTPUT_DIR = $(ROOT_DIR)/_output
BIN_DIR = $(OUTPUT_DIR)/bin

GO_BUILD := go build

CMDS := $(shell find $(ROOT_DIR)/cmd -mindepth 1 -maxdepth 1 -type d)

build: clean
	@for CMD in $(CMDS); do \
		echo build $$CMD; \
		$(GO_BUILD) -o $(BIN_DIR)/`basename $${CMD}` $$CMD; \
	done

clean:
	@rm -rf $(OUTPUT_DIR)

.PHONY: build clean
