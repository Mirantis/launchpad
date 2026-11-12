**Breaking: WinRM hosts without a complete credential are now rejected at config load.**

`launchpad` now runs rig's own connection validation for every host when the configuration is loaded. A `winRM` host with a `user` but no `password`, or with a password that is empty after environment substitution, now fails with `user and password must both be set for password authentication` instead of loading. A host entry with no connection block at all fails with `no protocol configuration`. Certificate authentication (`certPath` plus `keyPath`) is unaffected.

Such a configuration could not authenticate before this change either — it sent an HTTP Basic header with an empty password and failed at connect. The change is that it now fails at load, with a message naming the cause.

Rejection was chosen over substituting a default or warning and continuing, because the failure it replaces was silent: a password that vanished during YAML parsing left launchpad authenticating as the configured user with **no password at all**, against a host whose TLS certificate it may also not have verified. A warning would have been missed in exactly the runs where it mattered.

If you are affected, supply the password. Note that values are substituted into the configuration and then parsed as YAML, so a literal `$` must be written `$$` and a value containing an unquoted `#` is truncated at the `#` — which is how a password silently becomes empty. Full guidance: [Keeping credentials out of launchpad.yaml](https://github.com/Mirantis/launchpad/blob/main/docs/usage/getting-started.md).
