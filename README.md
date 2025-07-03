# Futura Monorepo

This monorepo contains all the code to make the OpisVigilant platform working with the except of the infrastructure

## Nix Development setup

This project uses `nix` to make it more portable. To install nix, you can run the following command `curl -fsSL https://install.determinate.systems/nix | sh -s -- install --determinate`.

### .env file

You need to create a `.env.local` file where you add secrets that shouldn't be part of the git commit. Nix will try to read the file and stop in case it cannot find it. A `.env.tmp` file is committed with the variables that need to be used.

### Run the shell

Assuming `nix` has been installed correctly and the `.env.local` is available, you can proceed with `nix develop` to enter the environment.

### Accessing Nats

Nats is deployed in Kubernetes using kind. The nix shell has the CLI installed. When the Nix shell starts, it runs a few commands against Nats which are described
in the `/scripts/setup-kv.sh`. Specifically, it will create the `api_keys` bucket and add an api key for testing.
To access the server run the following commands in two separate shells

```bash
# Shell 1
kubectl port-forward svc/nats 14222:4222
```

```bash
# from the Nix shell
nats kv get api_keys df9166bbacd761c74aecc50bb7a902342dd61a1de84551e253f7133154947d88 --server localhost:14222
# assuming the API Key is correct, you will see the following
api_keys > df9166bbacd761c74aecc50bb7a902342dd61a1de84551e253f7133154947d88 revision: 1 created @ 03 Jul 25 17:44 UTC

{"customer_id":"1", "status":"active", "cluster_id":"1", "cloud_provider_id":"1"}
```

For more commands, check this [page](https://docs.nats.io/nats-concepts/jetstream/key-value-store/kv_walkthrough).
