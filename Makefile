PROTO_DIR=proto/events
OUT_DIR=proto/gen
PROTOC_GEN_GO=$(shell which protoc-gen-go)
PROTOC_GEN_GO_GRPC=$(shell which protoc-gen-go-grpc)
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

##@ Proto 

.PHONY: proto-events
proto-events: proto-clean
	@echo "Generating shared events protos..."
	mkdir -p $(OUT_DIR)
	protoc --proto_path=proto --go_out=$(OUT_DIR) --go_opt=paths=source_relative $(wildcard $(PROTO_DIR)/*.proto)


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
PHONY: run-watcher
run-watcher:
	go run watcher/main.go

.PHONY: watcher-e2e
watcher-e2e:
	$(MAKE) -C watcher e2e

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