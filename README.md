# Futura Monorepo

This monorepo contains all the code to make the OpisVigilant platform working with the except of the infrastructure

## Workspaces

This is a monorepo containing multiple go modules. To start with, run the following make command

```make
$: make dev-env
```

## Frontend

You need to have `bun` installed in your system

## Kubernetes

Spin up a `kind` cluster and deploy the manifests that are in the `kind` folder of this project. You also need to have `ko` installed in the system to quickly run the projects in the `kind` cluster.

Make sure to have run `docker login` to push images to the registry.
