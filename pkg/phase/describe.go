package phase

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// Describe shows information about the current status of the cluster
type Describe struct {
	BasicPhase

	Ucp bool
	Dtr bool
}

// Title for the phase
func (p *Describe) Title() string {
	return "Display cluster status"
}

// Run does the actual saving of the local state file
func (p *Describe) Run() error {
	if p.Ucp {
		p.ucpReport()
	} else if p.Dtr {
		p.dtrReport()
	} else {
		p.hostReport()
	}

	return nil
}

func (p *Describe) ucpReport() {
	if !p.config.Spec.Ucp.Metadata.Installed {
		fmt.Println("Not installed")
		return
	}
	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t\n", "VERSION", "ADMIN_UI")
	uv := p.config.Spec.Ucp.Metadata.InstalledVersion
	urls := p.config.Spec.WebURLs()

	fmt.Fprintf(w, "%s\t%s\t\n", uv, urls.Ucp)
	w.Flush()
}

func (p *Describe) dtrReport() {
	if p.config.Spec.Dtr == nil || !p.config.Spec.Dtr.Metadata.Installed {
		fmt.Println("Not installed")
		return
	}

	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t\n", "VERSION", "ADMIN_UI")
	uv := p.config.Spec.Dtr.Metadata.InstalledVersion
	urls := p.config.Spec.WebURLs()

	fmt.Fprintf(w, "%s\t%s\t\n", uv, urls.Dtr)
	w.Flush()
}

func (p *Describe) hostReport() {
	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n", "ADDRESS", "INTERNAL_IP", "HOSTNAME", "ROLE", "OS", "ENGINE")

	for _, h := range p.config.Spec.Hosts {
		ev := "n/a"
		os := "n/a"
		ia := "n/a"
		hn := "n/a"
		if h.Metadata != nil {
			if h.Metadata.EngineVersion != "" {
				ev = h.Metadata.EngineVersion
			}
			if h.Metadata.Os != nil {
				os = fmt.Sprintf("%s/%s", h.Metadata.Os.ID, h.Metadata.Os.Version)
			}
			if h.Metadata.InternalAddress != "" {
				ia = h.Metadata.InternalAddress
			}
			if h.Metadata.Hostname != "" {
				hn = h.Metadata.Hostname
			}
		}
		fmt.Fprintf(w,
			"%s\t%s\t%s\t%s\t%s\t%s\t\n",
			h.Address,
			ia,
			hn,
			h.Role,
			os,
			ev,
		)
	}
	w.Flush()
}
