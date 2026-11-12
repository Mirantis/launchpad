**The Windows MCR install script must now be fetched over a transport that
can authenticate the server.** `spec.mcr.installURLWindows` is accepted only
as an `https://` URL or a local file; any other scheme is rejected instead
of being passed to `http.Get`, a redirect off https is refused, and a
non-200 response aborts rather than being stored as the script and uploaded
to the hosts.

**The Windows MCR install script's signature is now enforced.** launchpad
ran it under `-ExecutionPolicy Bypass`, which disables the Authenticode check
PowerShell would otherwise perform, so the signature Mirantis applies to
`install.ps1` was never verified on the host. The install and uninstall
invocations now use `-ExecutionPolicy AllSigned`, and PowerShell refuses a
script whose signature is missing, broken, or from an untrusted publisher.

`spec.mcr.installURLWindowsAllowUnsigned` restores the previous behaviour for
operators who point `installURLWindows` at a script of their own, which will
not generally carry a signature the host trusts. It is opt-in, named for what
it gives up, and applies to the uninstall invocation too, which runs the same
script with `-Uninstall`.

No checksum validation is applied to the install script. An earlier revision
of this change added a `spec.mcr.installURLWindowsChecksum` field; it was
dropped at the user's request, because the scripts are signed -- that
signature is the integrity anchor and a second digest adds nothing -- and
because `installURLWindows` is deliberately overridable for operators who
mirror or patch the script, which a pinned digest would conflict with. The
signature requirement takes the same objection seriously by making the
protection configurable alongside the value it protects, rather than
rejecting the configured case outright.
