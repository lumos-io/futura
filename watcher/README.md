# Watcher

This project is designed to subscribe to the Kubernetes Events and also collect eBPF signals and push them to either a webhook or to the console for debuggig.

## Getting started

We are using `Lima VM` for it. Validate that it is installed by running `lima --version`. If not, run `brew install lima` to install it. After that, start the VM with the below since we need to use it for the eBPF.

```shell
$: limactl start --name=watcher ./lima-ubuntu-vm.yaml
```

Select the option `Proceed with the current configuration` and once it is all done, run `limactl shell watcher` to ssh into the VM. Go to the folder `/tmp/code/futura` and then run `make dev-env` to prepare the go workspace.

To run the code, you need `sudo` since the eBPF requires elevated priviliges. So, `sudo go run main.go` and that's it. For your convenience, there is a Make rule that can help (`run`). You can run `sudo make run` for simplicity. Happy Coding!
