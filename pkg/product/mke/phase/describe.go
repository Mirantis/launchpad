package phase

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Mirantis/mcc/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// Describe shows information about the current status of the cluster.
type Describe struct {
	phase.BasicPhase

	MKE bool
	MSR bool
}

// Title for the phase.
func (p *Describe) Title() string {
	return "Display cluster status"
}

// Run does the actual saving of the local state file.
func (p *Describe) Run() error {
	if p.MKE {
		p.mkeReport()
	} else if p.MSR {
		p.msrReport()
	} else {
		p.hostReport()
	}

	return nil
}

func (p *Describe) mkeReport() {
	if !p.Config.Spec.MKE.Metadata.Installed {
		fmt.Println("Not installed")
		return
	}
	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t\n", "VERSION", "ADMIN_UI")
	uv := p.Config.Spec.MKE.Metadata.InstalledVersion
	mkeurl := "n/a"

	if url, err := p.Config.Spec.MKEURL(); err != nil {
		log.Debug(err)
	} else {
		mkeurl = url.String()
	}

	fmt.Fprintf(w, "%s\t%s\t\n", uv, mkeurl)
	w.Flush()
}

func (p *Describe) msrReport() {
	msrLeader := p.Config.Spec.MSRLeader()
	if msrLeader == nil || msrLeader.MSRMetadata == nil || !msrLeader.MSRMetadata.Installed {
		fmt.Println("Not installed")
		return
	}

	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t\n", "VERSION", "ADMIN_UI")
	uv := msrLeader.MSRMetadata.InstalledVersion
	msrurl := "n/a"

	if url, err := p.Config.Spec.MSRURL(); err != nil {
		log.Debug(err)
	} else {
		msrurl = url.String()
	}

	fmt.Fprintf(w, "%s\t%s\t\n", uv, msrurl)
	w.Flush()
}

func (p *Describe) hostReport() {
	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n", "ADDRESS", "INTERNAL_IP", "HOSTNAME", "ROLE", "OS", "RUNTIME")

	for _, h := range p.Config.Spec.Hosts {
		ev := "n/a"
		os := "n/a"
		ia := "n/a"
		hn := "n/a"
		if h.Metadata != nil {
			if h.Metadata.MCRVersion != "" {
				ev = h.Metadata.MCRVersion
			}
			if h.OSVersion.ID != "" {
				os = fmt.Sprintf("%s/%s", h.OSVersion.ID, h.OSVersion.Version)
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
			h.Address(),
			ia,
			hn,
			h.Role,
			os,
			ev,
		)
	}
	w.Flush()
}
