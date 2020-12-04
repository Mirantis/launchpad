package phase

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// Describe shows information about the current status of the cluster
type Describe struct {
	BasicPhase
}

// Title for the phase
func (p *Describe) Title() string {
	return "Display cluster status"
}

// Run does the actual saving of the local state file
func (p *Describe) Run() error {
	p.hostReport()

	return nil
}

func (p *Describe) hostReport() {
	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "ADDRESS", "PORT", "PROTOCOL", "CONNECTED")

	for _, h := range p.Config.Spec.Hosts {
		port := "n/a"
		proto := "n/a"
		if h.SSH != nil {
			port = fmt.Sprintf("%d", h.SSH.Port)
			proto = "SSH"
		} else if h.WinRM != nil {
			port = fmt.Sprintf("%d", h.WinRM.Port)
			proto = "WinRM"
		}
		fmt.Fprintf(w,
			"%s\t%s\t%s\t%t\n",
			h.Address,
			port,
			proto,
			h.Connection != nil,
		)
	}
	w.Flush()
}
