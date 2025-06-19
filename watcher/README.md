# Watcher

## Getting started

You can use `ko` to run the project in Kubernetes. After running `docker login` and be successfuly authenticated, do the following

```bash
KO_DOCKER_REPO=docker.io/davideberdin ko apply -f deploy/watcher.yaml
```
