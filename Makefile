GO_WORK_FILE=./go.work

PHONY: dev-env
dev-env:
ifeq ("$(wildcard $(GO_WORK_FILE))","")
	@echo "initialize go workspaces with Go 1.20 toolchain"
	GOTOOLCHAIN=go1.20+auto go work init
endif
	@echo "add all projects to go.work"
	go work use -r .
	go work sync

PHONY: run-watcher
run-watcher:
	go run watcher/main.go

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
.PHONY: watcher-generate
watcher-generate:
	$(MAKE) -C watcher generate

.PHONY: watcher-run
watcher-run: watcher-generate
	$(MAKE) -C watcher run

##@ Watcher Docker

.PHONY: watcher-docker-build
watcher-docker-build: 
	$(MAKE) -C watcher docker-build

.PHONY: watcher-docker-push
watcher-docker-push: 
	$(MAKE) -C watcher docker-push