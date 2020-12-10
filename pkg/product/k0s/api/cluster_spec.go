package api

// ClusterSpec defines cluster spec
type ClusterSpec struct {
	Hosts Hosts     `yaml:"hosts" validate:"required,dive,min=1"`
	K0s   K0sConfig `yaml:"k0sConfig,omitempty"`
}
