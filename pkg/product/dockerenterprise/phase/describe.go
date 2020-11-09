package phase

import (
	"fmt"
	"github.com/Mirantis/mcc/pkg/phase"
	"os"
	"text/tabwriter"
)

// Describe shows information about the current status of the cluster
type Describe struct {
	phase.BasicPhase

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
	if !p.Config.Spec.Ucp.Metadata.Installed {
		fmt.Println("Not installed")
		return
	}
	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t\n", "VERSION", "ADMIN_UI")
	uv := p.Config.Spec.Ucp.Metadata.InstalledVersion
	ucpurl := "n/a"
	url, err := p.Config.Spec.UcpURL()
	if err != nil {
		ucpurl = url.String()
	}

	fmt.Fprintf(w, "%s\t%s\t\n", uv, ucpurl)
	w.Flush()
}

func (p *Describe) dtrReport() {
	if p.Config.Spec.Dtr == nil || !p.Config.Spec.Dtr.Metadata.Installed {
		fmt.Println("Not installed")
		return
	}

	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t\n", "VERSION", "ADMIN_UI")
	uv := p.Config.Spec.Dtr.Metadata.InstalledVersion
	dtrurl := "n/a"
	url, err := p.Config.Spec.DtrURL()
	if err != nil {
		dtrurl = url.String()
	}

	fmt.Fprintf(w, "%s\t%s\t\n", uv, dtrurl)
	w.Flush()
}

func (p *Describe) hostReport() {
	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n", "ADDRESS", "INTERNAL_IP", "HOSTNAME", "ROLE", "OS", "ENGINE")

	for _, h := range p.Config.Spec.Hosts {
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
