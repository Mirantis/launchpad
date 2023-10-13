package msr

// TODO(squizzi): Figure out a way to build these automatically from
// bootstrappers

// SharedInstallJoinFlags are flags which are shared amongst the join and
// install bootstrap commands.
// The --replica-id and --ucp-node are shared flags, but their values shouldn't
// be shared so they are not included here.
var SharedInstallJoinFlags = []string{
	"--debug",
	"--skip-network-test",
	"--replica-https-port",
	"--replica-http-port",
	"--replica-rethinkdb-cache-mb",
	"--ucp-ca",
	"--ucp-insecure-tls",
}

// SharedInstallUpgradeFlags are flags which are shared amongst the install and
// upgrade bootstrap commands.
var SharedInstallUpgradeFlags = []string{
	"--debug",
	"--ucp-ca",
	"--ucp-insecure-tls",
}

// SharedInstallRemoveFlags are lfags which are shared amongst the install and
// remove bootstrap commands.
var SharedInstallRemoveFlags = []string{
	"--debug",
	"--ucp-ca",
	"--ucp-insecure-tls",
}
