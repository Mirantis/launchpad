package api

// ClusterSpec defines cluster spec
type ClusterSpec struct {
	Hosts Hosts `yaml:"hosts" validate:"required,dive,min=1"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type spec ClusterSpec
	yc := (*spec)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	return nil
}
