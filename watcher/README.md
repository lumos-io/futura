# Watcher

## Getting started

You can use `ko` to run the project in Kubernetes. After running `docker login` and be successfuly authenticated, do the following

```bash
KO_DOCKER_REPO=docker.io/davideberdin ko apply -f deploy/watcher.yaml
```

### Lima VM

We are using `Lima VM` for it. Validate that it is installed by running `lima --version`. If not, run `brew install lima` to install it. After that, start the VM with the below since we need to use it for the eBPF.

```shell
$: limactl start --name=watcher ./lima-ubuntu-vm.yaml
```

Select the option `Proceed with the current configuration` and once it is all done, run `limactl shell watcher` to ssh into the VM. Go to the folder `/tmp/code/futura` and then run `make dev-env` to prepare the go workspace. Remember to run `export PATH="/usr/local/go/bin:${PATH}"` in the shell otherwise you cannot run `go`.

To run the code, you need `sudo` since the eBPF requires elevated priviliges. So, `sudo go run main.go` and that's it. For your convenience, there is a Make rule that can help (`run`). You can run `sudo make run` for simplicity. Happy Coding!
