package v1beta2

// ClusterMeta defines cluster metadata
type ClusterMeta struct {
	Name string `yaml:"name" validate:"required"`
}

// ClusterConfig describes cluster.yaml configuration
type ClusterConfig struct {
	APIVersion       string       `yaml:"apiVersion" validate:"eq=launchpad.mirantis.com/v1beta2"`
	Kind             string       `yaml:"kind" validate:"eq=UCP"`
	Metadata         *ClusterMeta `yaml:"metadata"`
	Spec             *ClusterSpec `yaml:"spec"`
	ManagerJoinToken string       `yaml:"-"`
	WorkerJoinToken  string       `yaml:"-"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	c.Metadata = &ClusterMeta{
		Name: "launchpad-ucp",
	}
	c.Spec = &ClusterSpec{}

	type yclusterconfig ClusterConfig
	yc := (*yclusterconfig)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	return nil
}
