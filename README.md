# Futura Monorepo

This monorepo contains all the code to make the OpisVigilant platform working with the except of the infrastructure

## Nix Development setup

This project uses `nix` to make it more portable. To install nix, you can run the following command `curl -fsSL https://install.determinate.systems/nix | sh -s -- install --determinate`.

### .env file

You need to create a `.env.local` file where you add secrets that shouldn't be part of the git commit. Nix will try to read the file and stop in case it cannot find it. A `.env.tmp` file is committed with the variables that need to be used.

### Run Nix

Assuming `nix` has been installed correctly and the `.env.local` is available, you can proceed with `nix develop` to enter the environment. By default, `kind` is not created (also metric-server and redis are skipped). To enable Kind deployment, run nix as follow `SKIP_KIND=false nix develop`.

#### Accessing Redis

Redis is deployed in Kubernetes using kind. The nix shell has the CLI installed. When the Nix shell starts, it runs a few commands against Redis which are described
in the `/scripts/setup-kv.sh`. Specifically, it will create the `api_keys` bucket and add an api key for testing.
To access the server run the following commands in two separate shells

```bash
# Shell 1
kubectl port-forward svc/redis 16379:6379
```

and then to access redis, in the second terminal you can use the `redis-cli` to connect and run commands.

## Docker Compose Profile

Now all the containers need to be created for local development. I have attached a `profile` flag to the containers so that they will be created if and only if the profile is specified. For example

```bash
docker compose --profile dev-tools up -d
```

will also start the containers with that profile (like `redisinsight`).
