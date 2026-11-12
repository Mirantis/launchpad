**Breaking: `spec.mke.adminPassword` is now required.**

A configuration that omits it stops applying, with:

```
failed to validate cluster config: cluster config validation failed: Key: 'ClusterConfig.Spec.MKE.AdminPassword' Error:Field validation for 'AdminPassword' failed on the 'required' tag
```

Set `spec.mke.adminPassword`, or pass `--admin-password` in `spec.mke.installFlags` or `spec.mke.upgradeFlags`; the flag form is hoisted into the field before validation, so either satisfies the requirement. Supplying both the field and the flag with different values is still rejected.

Previously, omitting the password let the MKE bootstrapper generate one. Launchpad then recovered that password by matching a regular expression against the installer's output and logged it in the clear at warning level, because every later phase that authenticates to MKE — client bundle download, the MKE health check, node removal — needs it and there was no other source. The generated password was also written unmasked to the install log, since no value existed to register as a redaction before the installer ran. Requiring the password removes both the log exposure and the dependency on the installer's output format, which launchpad does not control; defaulting it or continuing to scrape it would have kept one or the other.
