# project-a Makefile

PROJECT_NAME=pipeline
SERVICE_PROTO_DIR=proto/services
SERVICE_OUT_DIR=proto/gen/services
SHARED_PROTO_DIR=../proto
PROTOC_GEN_GO=$(shell which protoc-gen-go)
PROTOC_GEN_GO_GRPC=$(shell which protoc-gen-go-grpc)

SERVICE_PROTOS=$(wildcard $(SERVICE_PROTO_DIR)/*.proto)

.PHONY: proto-generate proto-clean

proto-generate: proto-clean
	@echo "Generating service protos for $(PROJECT_NAME)..."
	mkdir -p $(SERVICE_OUT_DIR)
	protoc \
		--proto_path=$(SERVICE_PROTO_DIR) \
		--proto_path=$(SHARED_PROTO_DIR) \
		--go_out=$(SERVICE_OUT_DIR) \
		--go-grpc_out=$(SERVICE_OUT_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		$(SERVICE_PROTOS)

proto-clean:
	rm -rf $(SERVICE_OUT_DIR)
