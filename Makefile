PROTO_DIR=proto
OUT_DIR=proto/gen

PROTO_BACKEND_DIR=proto/backend
OUT_BACKEND_DIR=proto/gen/backend

PROTOC_GEN_GO=$(shell which protoc-gen-go)
PROTOC_GEN_GO_GRPC=$(shell which protoc-gen-go-grpc)
PROTOC_GEN_TS_PROTO=$(shell which protoc-gen-ts_proto)

PROTO_FILES := $(shell find $(PROTO_DIR) -name '*.proto')
PROTO_BACKEND_FILES := $(shell find $(PROTO_BACKEND_DIR) -name '*.proto')

GO_WORK_FILE=./go.work

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
proto-files: proto-clean proto-go proto-ts

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


##@ APIs
.PHONY: apis-run
apis-run: frontend-build
	$(MAKE) -C apis run