**Go 1.27.4 and a dependency refresh.**

The toolchain moves from 1.27.3 to 1.27.4, which carries the fix for
GO-2026-6859 — an HTTP/1 server crash from a `MaxBytesReader` header race in
`net/http`. `govulncheck` reported it against 26 call sites in this repository
and now reports no vulnerabilities.

Dependencies are updated alongside it, most notably Kubernetes 0.36.2 to
0.37.2 and Helm 3.21.4 to 3.22.1. No launchpad source changed.
