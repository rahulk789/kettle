BINARY=kettle
PROTO_DIR=./api/shim
OUT_DIR=./api/shim
PROTOC_GEN_TTRPC=$(shell which protoc-gen-ttrpc)

build-all:
	go build -o $(BINARY)

# Generate Go code from proto files for ttrpc
generate-proto:
	@mkdir -p $(OUT_DIR)
	@find $(PROTO_DIR) -name "*.proto" | while read proto_file; do \
		protoc \
			-I=$(PROTO_DIR) \
			--go-ttrpc_out=$(OUT_DIR) \
      --go_out=$(OUT_DIR) \
			$$proto_file; \
	done

.PHONY: check-tools generate-proto build clean

