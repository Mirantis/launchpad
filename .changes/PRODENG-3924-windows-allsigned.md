**Windows MCR install failures now say why PowerShell refused the script.**

rig truncates a command's output into its error at roughly a hundred
characters, so a refusal arrived as `File C:\...\installerWindows123.ps1
cannot be loaded. The contents of file  ...` — cut off exactly where the
reason would be. An install failure now carries the installer's own output,
the script's `Get-AuthenticodeSignature` status and signer, and the host's
effective execution policies.

**`spec.mcr.installURLWindowsAllowUnsigned` is exposed by the aws-simple
example**, as `launchpad.mcr_install_url_windows_allow_unsigned`. It
defaults to `false`, matching launchpad's own default.

The smoke suites set it to `true`. Enforcing `-ExecutionPolicy AllSigned`
causes a stock Windows host to refuse the signed script published at
get.mirantis.com, so every Windows host fails at Install MCR before any
launchpad behaviour is exercised. The opt-out keeps the suites meaningful
while the refusal is diagnosed.
