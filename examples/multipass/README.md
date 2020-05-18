# Bootstrapping UCP cluster with Multipass

This directory provides an example flow with mcc tool together with [Multipass](http://multipass.run/).

## Steps

1. Create machines `multipass launch -n manager --mem 4G --disk 10G && multipass launch -n worker --disk 10G`
2. Update IP addresses in `cluster.yml` (run `multipass ls` to find correct IPs)
3. `sudo ./mcc install -c ./examples/multipass/cluster.yml` (access to ssh key file requires sudo)
4. Profit! :)
