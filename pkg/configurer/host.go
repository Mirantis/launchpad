package configurer

import (
	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/cmd"
	"github.com/k0sproject/rig/v2/remotefs"
)

// Host is the interface that configurer methods use to interact with a remote
// host. It is satisfied by *rig.Client (and therefore by the product host
// types, which embed rig.CompositeConfig and *rig.Client).
type Host interface {
	cmd.SimpleRunner
	// ContextRunner is needed by configurers that must bound an exec call
	// with an explicit deadline (see pkg/configurer/windows.go's
	// windowsExecTimeout) rather than rig's default context.Background().
	cmd.ContextRunner
	Sudo() *rig.Client
	FS() remotefs.FS
}
