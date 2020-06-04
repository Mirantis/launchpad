# Testing mcc with footloose

[Footloose](https://github.com/weaveworks/footloose) creates containers that look like virtual machines.

## Steps

1. `footloose create`
2. `launchpad apply`
3. Profit :)

## Note
If you are using Docker Desktop make sure that the Kubernetes feature is OFF.
When `footloose` spins up your UCP cluster it will automatically forward port 6443 on your host to port 6443 on your UCP manager.
If you are running Kubernetes locally via Docker Desktop this will fail, because port 6443 is already in use.
