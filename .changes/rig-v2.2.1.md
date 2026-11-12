**rig is taken from upstream again.**

`go.mod` carried `replace github.com/k0sproject/rig/v2 =>
github.com/james-nesbitt/rig/v2`, pinned to a September pre-release of a
fork. The fork existed to carry one fix — bounding WinRM `command.Wait()`
and `remotefs` upload against a dead session, which had hung
`smoke-windows` for 40+ minutes — that upstream had not yet merged.

Upstream merged it (k0sproject/rig#473) and released it. The replace is
removed and launchpad builds against `github.com/k0sproject/rig/v2 v2.2.1`,
which carries that fix plus roughly fifty commits of SSH, remotefs and
WinRM fixes made since the fork point.
