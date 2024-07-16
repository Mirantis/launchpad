package phase

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Mirantis/mcc/pkg/msr/msr3"
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
	// Configure the columns to start.
	tabWriter := new(tabwriter.Writer)
	// minwidth, tabwidth, padding, padchar, flags
	tabWriter.Init(os.Stdout, 8, 8, 1, '\t', 0)
	fmt.Fprintf(tabWriter, "%s\t%s\t\n", "VERSION", "ADMIN_UI")

	// Populate potential MSR2 entries.
	msr2Leader := p.Config.Spec.MSR2Leader()
	installedVersion := msr2Leader.MSR2Metadata.InstalledVersion

	msrURL := "n/a"

	if msr2Leader == nil || msr2Leader.MSR2Metadata == nil || !msr2Leader.MSR2Metadata.Installed {
		// If no MSR2 is installed just don't add anything to the writer.
		log.Debugf("No MSR2 products found")
	} else {
		if url, err := p.Config.Spec.MSR2URL(); err != nil {
			log.Infof("failed to get MSR URL: %s", err)
		} else {
			msrURL = url.String()
		}

		fmt.Fprintf(tabWriter, "%s\t%s\t\n", installedVersion, msrURL)
	}

	var err error

	msr3s := p.Config.Spec.MSR3s()
	msr3Leader := msr3s.First()
	installedVersion = msr3Leader.MSR3Metadata.InstalledVersion

	if msr3Leader == nil || msr3Leader.MSR3Metadata == nil || !msr3Leader.MSR3Metadata.Installed {
		// If no MSR3 is installed just don't add anything to the writer.
		log.Debugf("No MSR3 products found")
	} else {
		// Populate potential MSR3 entries.
		msrURL, err = msr3.GetMSRURL(p.Config)
		if err != nil {
			log.Infof("failed to get MSR URL: %s", err)
		}
	}

	fmt.Fprintf(tabWriter, "%s\t%s\t\n", installedVersion, msrURL)

	// Flush the added entries to the writer.
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
