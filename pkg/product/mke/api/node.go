package api

import (
	"time"
)

const (
	NODE_READY_STATE = "ready"
)

// NodeDescriptionEnginePlugin node description struct for the engine plugin
type NodeDescriptionEnginePlugin struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
}

// NodeDescriptionEngine node description struct
type NodeDescriptionEngine struct {
	EngineVersion string                        `json:"engine_version" yaml:"engine_version"`
	Labels        []string                      `json:"labels" yaml:"labels"`
	Plugins       []NodeDescriptionEnginePlugin `json:"plugins" yaml:"plugins"`
}

// NodeDescriptionPlatform struct containing the platform description of the node
type NodeDescriptionPlatform struct {
	Architecture string `json:"architecture" yaml:"architecture"`
	OS           string `json:"os" yaml:"os"`
}

// NodeDescriptionDiscreteGenericResource node struct for a discrete k8s resource
type NodeDescriptionDiscreteGenericResource struct {
	Kind  string `json:",omitempty" yaml:",omitempty"`
	Value int64  `json:",omitempty" yaml:",omitempty"`
}

// NodeDescriptionNamedGenericResource node struct for a generic k8s resource
type NodeDescriptionNamedGenericResource struct {
	Kind  string `json:",omitempty" yaml:",omitempty"`
	Value string `json:",omitempty" yaml:",omitempty"`
}

// NodeDescriptionGenericResource struct containing the different resources of the node
type NodeDescriptionGenericResource struct {
	NamedResourceSpec    NodeDescriptionNamedGenericResource    `json:",omitempty" yaml:",omitempty"`
	DiscreteResourceSpec NodeDescriptionDiscreteGenericResource `json:",omitempty" yaml:",omitempty"`
}

// NodeDescriptionResources struct containing description of the node resources
type NodeDescriptionResources struct {
	GenericResources []NodeDescriptionGenericResource
	MemoryBytes      int64 `yaml:"memory_bytes"`
	NanoCPUs         int64 `yaml:"nano_cpus"`
}

// NodeDescriptionTLSInfo struct containing the TLS Info of the node
type NodeDescriptionTLSInfo struct {
	TrustRoot string `json:",omitempty" yaml:",omitempty"`
	// CertIssuer is the raw subject bytes of the issuer
	CertIssuerSubject string `json:",omitempty" yaml:",omitempty"`
	// CertIssuerPublicKey is the raw public key bytes of the issuer
	CertIssuerPublicKey string `json:",omitempty" yaml:",omitempty"`
}

// NodeDescription struct containing the description of the node
type NodeDescription struct {
	Engine    NodeDescriptionEngine    `json:"engine_description" yaml:"engine_description"`
	Hostname  string                   `json:"hostname" yaml:"hostname"`
	Platform  NodeDescriptionPlatform  `json:"platform" yaml:"platform"`
	Resources NodeDescriptionResources `json:"resources" yaml:"resources"`
	TLSInfo   NodeDescriptionTLSInfo   `json:"tls_info" yaml:"tls_info"`
}

// NodeManagerStatus struct containing the node manager status
type NodeManagerStatus struct {
	Addr         string `json:"addr" yaml:"addr"`
	Leader       bool   `json:"leader" yaml:"leader"`
	Reachability string `json:"reachability" yaml:"reachability"`
}

// NodeSpec struct containing the node k8s spec
type NodeSpec struct {
	Availability string   `json:"availability" yaml:"availability"`
	Labels       []string `json:"labels" yaml:"labels"`
	Name         string   `json:"name" yaml:"name"`
	Role         string   `json:"role" yaml:"role"`
}

// NodeStatus struct containing the status of the node
type NodeStatus struct {
	Addr    string `json:"addr" yaml:"addr"`
	Message string `json:"message" yaml:"message"`
	State   string `json:"state" yaml:"state"`
}

// NodeObjectVersion struct containing the node object uint64 index
type NodeObjectVersion struct {
	Index uint64 `json:"index" yaml:"index"`
}

// Node contains representation of the node object returned by the /nodes API call
type Node struct {
	CreatedAt     time.Time         `json:"created_at" yaml:"created_at"`
	Description   NodeDescription   `json:"description" yaml:"description"`
	ID            string            `json:"id" yaml:"id"`
	ManagerStatus NodeManagerStatus `json:"node_manager_status" yaml:"node_manager_status"`
	Spec          NodeSpec          `json:"node_spec" yaml:"node_spec"`
	Status        NodeStatus        `json:"status" yaml:"status"`
	UpdatedAt     time.Time         `json:"updated_at" yaml:"updated_at"`
	Version       NodeObjectVersion `json:"version" yaml:"version"`
}

// IsReady returns if node is in 'ready' state
func (n *Node) IsReady() bool {
	return n.Status.State == NODE_READY_STATE
}
