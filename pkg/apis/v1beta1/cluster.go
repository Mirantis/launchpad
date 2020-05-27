package v1beta1

// ClusterMeta defines cluster metadata
type ClusterMeta struct {
	Name string `yaml:"name" validate:"required"`
}

// ClusterConfig describes cluster.yaml configuration
type ClusterConfig struct {
	APIVersion       string       `yaml:"apiVersion" validate:"eq=launchpad.mirantis.com/v1beta1"`
	Kind             string       `yaml:"kind" validate:"eq=UCP"`
	Metadata         *ClusterMeta `yaml:"metadata"`
	Spec             *ClusterSpec `yaml:"spec"`
	ManagerJoinToken string       `yaml:"-"`
	WorkerJoinToken  string       `yaml:"-"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawClusterConfig ClusterConfig
	raw := rawClusterConfig{
		Metadata: &ClusterMeta{
			Name: "mcc-ucp",
		},
		Spec: &ClusterSpec{},
	}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*c = ClusterConfig(raw)
	return nil
}
