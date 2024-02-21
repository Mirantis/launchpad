# MKEx configurers

Here we have configurers meant to allow launchpad to run with the Mirantis MKEx providing 
platforms.

Note that at least some portion of these configurers use dummy functionality for some
behaviour in order to make launchpad compatible with the MKEx idea of installing MKE 
without installing MCR, as it will already be installed in the base OS.

## Resolution

Resolution is handled by injecting a new linux resolver into the rig.Resolvers list,
which can handle rig.Connections for MKEx OSes by looking for the expected file system.
The injection is executed in the mkex/resolver.go init()

## OSes

### MKEx 

This is the base MKEx Rocky linux w/ ostree base OS from Mirantis/CIQ.

The OS is meant to allow runtime changes such as Docker operations, and some global 
configuration changes, but should not run any system changes such as package management.

User management should be unnecesary as well.
