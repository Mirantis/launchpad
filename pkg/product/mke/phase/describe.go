package phase

import (
	"fmt"
	"os"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"

	"github.com/Mirantis/mcc/pkg/phase"
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
	switch {
	case p.MKE:
		p.mkeReport()
	case p.MSR:
		p.msrReport()
	default:
		p.hostReport()
	}

	return nil
}

func (p *Describe) mkeReport() {
	if !p.Config.Spec.MKE.Metadata.Installed {
		fmt.Println("Not installed")
		return
	}
	tabWriter := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	tabWriter.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(tabWriter, "%s\t%s\t\n", "VERSION", "ADMIN_UI")
	uv := p.Config.Spec.MKE.Metadata.InstalledVersion
	mkeurl := "n/a"

	if url, err := p.Config.Spec.MKEURL(); err != nil {
		log.Debug(err)
	} else {
		mkeurl = url.String()
	}

	fmt.Fprintf(tabWriter, "%s\t%s\t\n", uv, mkeurl)
	tabWriter.Flush()
}

func (p *Describe) msrReport() {
	msrLeader := p.Config.Spec.MSRLeader()
	if msrLeader == nil || msrLeader.MSRMetadata == nil || !msrLeader.MSRMetadata.Installed {
		fmt.Println("Not installed")
		return
	}

	tabWriter := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	tabWriter.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(tabWriter, "%s\t%s\t\n", "VERSION", "ADMIN_UI")
	uv := msrLeader.MSRMetadata.InstalledVersion
	msrURL := "n/a"

	var err error

	switch p.Config.Spec.MSR.MajorVersion() {
	case 2:
		if url, err := p.Config.Spec.MSR2URL(); err != nil {
			log.Infof("failed to get MSR URL: %s", err)
		} else {
			msrURL = url.String()
		}
	case 3:
		msrURL, err = getMSRURL(p.Config)
		if err != nil {
			log.Infof("failed to get MSR URL: %s", err)
		}
	}

	fmt.Fprintf(tabWriter, "%s\t%s\t\n", uv, msrURL)
	tabWriter.Flush()
}

func (p *Describe) hostReport() {
	tabWriter := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	tabWriter.Init(os.Stdout, 8, 8, 1, '\t', 0)

	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\t%s\t\n", "ADDRESS", "INTERNAL_IP", "HOSTNAME", "ROLE", "OS", "RUNTIME")

	for _, h := range p.Config.Spec.Hosts {
		mcrV := "n/a"
		hostOS := "n/a"
		internalAddr := "n/a"
		hostname := "n/a"
		if h.Metadata != nil {
			if h.Metadata.MCRVersion != "" {
				mcrV = h.Metadata.MCRVersion
			}
			if h.OSVersion.ID != "" {
				hostOS = fmt.Sprintf("%s/%s", h.OSVersion.ID, h.OSVersion.Version)
			}
			if h.Metadata.InternalAddress != "" {
				internalAddr = h.Metadata.InternalAddress
			}
			if h.Metadata.Hostname != "" {
				hostname = h.Metadata.Hostname
			}
		}
		fmt.Fprintf(tabWriter,
			"%s\t%s\t%s\t%s\t%s\t%s\t\n",
			h.Address(),
			internalAddr,
			hostname,
			h.Role,
			hostOS,
			mcrV,
		)
	}
	tabWriter.Flush()
}
