PROTO_DIR=proto
OUT_DIR=proto/gen

PROTO_BACKEND_DIR=proto/backend
PROTO_ENGINE_DIR=proto/engine
OUT_BACKEND_DIR=proto/gen/backend
OUT_ENGINE_DIR=proto/gen/engine

PROTOC_GEN_GO=$(shell which protoc-gen-go)
PROTOC_GEN_GO_GRPC=$(shell which protoc-gen-go-grpc)
PROTOC_GEN_TS_PROTO=$(shell which protoc-gen-ts_proto)

PROTO_FILES := $(shell find $(PROTO_DIR) -name '*.proto')
PROTO_BACKEND_FILES := $(shell find $(PROTO_BACKEND_DIR) -name '*.proto')
PROTO_ENGINE_FILES := $(shell find $(PROTO_ENGINE_DIR) -name '*.proto')

GO_WORK_FILE=./go.work

##@ Testing

.PHONY: test
test: test-go test-python test-frontend ## Run all tests in the project

.PHONY: test-go
test-go: ## Run all Go service tests
	@echo "==> Running Go tests..."
	@echo "  - Testing APIs..."
	@cd apis && go test -v -race ./... || exit 1
	@echo "  - Testing Analytics (requires ClickHouse running)..."
	@cd analytics && go test -v -race ./... || exit 1
	@echo "  - Testing Pipeline..."
	@cd pipeline && go test -v -race ./... || exit 1
	@echo "  - Testing Watcher..."
	@cd watcher && go test -v -race $(shell cd watcher && go list ./... | grep -v '/internal/ebpf/bpf/') || exit 1
	@echo "  - Testing Operator..."
	@cd operator && go mod tidy && make build || exit 1
	@echo "✅ All Go tests passed!"

.PHONY: setup-clickhouse-migrations
setup-clickhouse-migrations: ## Run ClickHouse migrations (requires ClickHouse running on localhost:9000)
	@echo "==> Running ClickHouse migrations..."
	@command -v migrate >/dev/null 2>&1 || { echo "Error: golang-migrate not installed. Install: brew install golang-migrate"; exit 1; }
	@echo "  - Creating events database..."
	@clickhouse-client --host localhost --query "CREATE DATABASE IF NOT EXISTS events" || echo "Note: If connection fails, check ClickHouse credentials in docker-compose.yaml"
	@echo "  - Running Analytics migrations..."
	@migrate -path analytics/db/migrations \
		-database "clickhouse://default@localhost:9000/events?x-multi-statement=true" \
		up
	@echo "  - Running Engine migrations..."
	@migrate -path engine/db/migrations \
		-database "clickhouse://default@localhost:9000/events?x-multi-statement=true" \
		up
	@echo "✅ ClickHouse migrations completed!"

.PHONY: test-python
test-python: ## Run Python engine tests
	@echo "==> Running Python tests..."
	@cd engine && uv sync && uv run python3 -m pytest tests/ -v || exit 1
	@echo "✅ Python tests passed!"

.PHONY: test-frontend
test-frontend: ## Run frontend lint and type-check
	@echo "==> Running Frontend tests..."
	@cd frontend && bun install && bun run lint --max-warnings 20 && bun run type-check || exit 1
	@echo "✅ Frontend tests passed!"

PHONY: dev-env
dev-env:
ifeq ("$(wildcard $(GO_WORK_FILE))","")
	@echo "initialize go workspaces with Go 1.24.1 toolchain"
	GOTOOLCHAIN=go1.24+auto go work init
endif
	@echo "add all projects to go.work"
	go work use -r .
	go work sync
	@echo "install TS dependencies for protos"
	cd proto && bun install

##@ Proto 
.PHONY: proto-files
proto-files: proto-clean proto-go proto-ts proto-py

.PHONY: proto-go
proto-go:
	@echo "Generating Go protos..."
	@find $(PROTO_DIR) -name "*.proto"
	mkdir -p $(OUT_DIR)
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(OUT_DIR) \
		--go-grpc_out=$(OUT_DIR) \
		--go-grpc_opt=paths=source_relative \
		--go_opt=paths=source_relative \
		$(PROTO_FILES)

.PHONY: proto-ts
proto-ts:
	@echo "Generating TypeScript protos..."
	@find $(PROTO_BACKEND_DIR) -name "*.proto"
	mkdir -p $(OUT_BACKEND_DIR)
	protoc --plugin=protoc-gen-ts=$(PROTOC_GEN_TS_PROTO) \
		--ts_out=$(OUT_BACKEND_DIR) \
		--ts_opt=snakeToCamel=false,esModuleInterop=true,useExactTypes=true,stringEnums=true,outputJsonMethods=true,paths=source_relative \
		--proto_path=$(PROTO_BACKEND_DIR) \
		$(PROTO_BACKEND_FILES)

.PHONY: proto-py
proto-py: ensure-proto-deps
	@echo "Generating Python protos..."
	@find $(PROTO_ENGINE_DIR) -name "*.proto"
	mkdir -p $(OUT_ENGINE_DIR)
	uv run -m grpc_tools.protoc -I=$(PROTO_ENGINE_DIR) \
		--python_out=$(OUT_ENGINE_DIR) \
		--pyi_out=$(OUT_ENGINE_DIR) \
		--grpc_python_out=$(OUT_ENGINE_DIR) \
		$(PROTO_ENGINE_FILES)
	touch $(OUT_ENGINE_DIR)/__init__.py

.PHONY: ensure-proto-deps
ensure-proto-deps:
	@echo "Checking Python deps for proto generation (inside proto/.venv)..."
	uv venv && \
	if ! uv pip show protobuf >/dev/null 2>&1; then \
		echo "Installing protobuf..."; \
		uv pip install protobuf; \
	fi && \
	if ! uv pip show grpcio >/dev/null 2>&1; then \
		echo "Installing grpcio..."; \
		uv pip install grpcio; \
	fi && \
	if ! uv pip show grpcio-tools >/dev/null 2>&1; then \
		echo "Installing grpcio-tools..."; \
		uv pip install grpcio-tools; \
	fi

.PHONY: proto-clean
proto-clean:
	rm -rf proto/gen

##@ Operator Build
.PHONY: operator-manifests
operator-manifests: 
	$(MAKE) -C operator manifests

.PHONY: operator-generate
operator-generate: 
	$(MAKE) -C operator generate

.PHONY: operator-fmt
operator-fmt: 
	$(MAKE) -C operator fmt

.PHONY: operator-vet
operator-vet: 
	$(MAKE) -C operator vet	

.PHONY: operator-test
operator-test:
	$(MAKE) -C operator test

.PHONY: operator-build
operator-build: operator-manifests operator-generate operator-fmt operator-vet 
	go build -o operator/bin/manager operator/cmd/main.go

.PHONY: operator-run
operator-run: operator-manifests operator-generate ## Run a controller from your host.
	go run operator/cmd/main.go

##@ Operator Docker

.PHONY: operator-docker-build
operator-docker-build: 
	$(MAKE) -C operator docker-build

.PHONY: operator-docker-buildx
operator-docker-buildx: 
	$(MAKE) -C operator docker-buildx	

.PHONY: operator-docker-push
operator-docker-push: 
	$(MAKE) -C operator docker-push

##@ Operator Install
##@ Make sure kind or minikube is running otherwise the command(s) will fail

.PHONY: operator-install
operator-install: 
	$(MAKE) -C operator install

.PHONY: operator-uninstall
operator-uninstall: 
	$(MAKE) -C operator uninstall

.PHONY: operator-deploy
operator-deploy: 
	$(MAKE) -C operator deploy

.PHONY: operator-undeploy
operator-undeploy: 
	$(MAKE) -C operator undeploy		

##@ Watcher Run
##@ Watcher
.PHONY: watcher-deploy
watcher-deploy:
	$(MAKE) -C watcher deploy

.PHONY: watcher-run
watcher-run:	
	$(MAKE) -C watcher run

##@ Watcher Docker

.PHONY: watcher-docker-build
watcher-docker-build: 
	$(MAKE) -C watcher docker-build

.PHONY: watcher-docker-push
watcher-docker-push: 
	$(MAKE) -C watcher docker-push

##@ Frontend

.PHONY: frontend-dev
frontend-dev:
	$(MAKE) -C frontend dev

.PHONY: frontend-build
frontend-build:
	$(MAKE) -C frontend build	

.PHONY: frontend-preview
frontend-preview:
	$(MAKE) -C frontend preview

.PHONY: frontend-lint
frontend-lint:
	$(MAKE) -C frontend lint

##@ Pipeline
.PHONY: pipeline-deploy
pipeline-deploy:
	$(MAKE) -C pipeline deploy

.PHONY: pipeline-run
pipeline-run:
	$(MAKE) -C pipeline run

##@ Analytics
.PHONY: analytics-run
analytics-run:
	$(MAKE) -C analytics run

##@ APIs
.PHONY: apis-run
apis-run: frontend-build
	$(MAKE) -C analytics run & \
	$(MAKE) -C apis run & \
	wait
##@ Help

.PHONY: help
help: ## Display this help message
	@echo "Futura Monorepo - Available Make Targets"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-20s\033[0m %s\n", "Target", "Description"} /^[a-zA-Z_-]+:.*?##/ { printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
