.PHONY: deploy
deploy:	
	KO_DOCKER_REPO=docker.io/davideberdin ko apply -f deploy/pipeline.yaml

.PHONY: collect
collect:
	go run main.go collect

.PHONY: validate
validate:
	go run main.go validate

.PHONY: enrich
enrich:
	go run main.go enrich	

.PHONY: store
store:
	go run main.go store	

.PHONY: all
all:
	go run main.go collect & \
	go run main.go validate & \
	go run main.go enrich & \
	go run main.go store & \
	wait