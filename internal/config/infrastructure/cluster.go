package infrastructure

import (
	"fmt"

	"github.com/qpubio/qpub-server/internal/shared/cluster"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/caarlos0/env/v11"
)

type Cluster struct {
	// Full identifiers from environment
	Name   string `env:"CLUSTER_NAME" envDefault:"core-dev-us-east-1a"`
	VMName string `env:"VM_NAME" envDefault:"core-d-us1a-api-01"`

	// Parsed cluster components
	ClusterScope  string
	ClusterEnv    string
	ClusterRegion string
	ClusterZone   string

	// Parsed VM components
	VMScope  string
	VMEnv    string
	VMRegion string
	VMZone   string
	VMRole   string
	VMSeq    string
}

func NewCluster() (*Cluster, error) {
	cfg := &Cluster{}

	// Parse environment variables
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse cluster config: %w", err)
	}

	// Parse cluster name
	clusterComponents, err := cluster.ParseClusterName(cfg.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cluster name: %w", err)
	}

	// Parse VM name
	vmComponents, err := cluster.ParseVMName(cfg.VMName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse VM name: %w", err)
	}

	// Store parsed cluster components
	cfg.ClusterScope = clusterComponents.Scope
	cfg.ClusterEnv = clusterComponents.Env
	cfg.ClusterRegion = clusterComponents.Region
	cfg.ClusterZone = clusterComponents.Zone

	// Store parsed VM components
	cfg.VMScope = vmComponents.Scope
	cfg.VMEnv = vmComponents.Env
	cfg.VMRegion = vmComponents.Region
	cfg.VMZone = vmComponents.Zone
	cfg.VMRole = vmComponents.Role
	cfg.VMSeq = vmComponents.Seq

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cluster config: %w", err)
	}

	return cfg, nil
}

func (c *Cluster) Validate() error {
	// Validate parsed cluster components
	if c.ClusterScope == "" {
		return fmt.Errorf("cluster scope cannot be empty")
	}
	if c.ClusterEnv == "" {
		return fmt.Errorf("cluster env cannot be empty")
	}
	if c.ClusterRegion == "" {
		return fmt.Errorf("cluster region cannot be empty")
	}
	if c.ClusterZone == "" {
		return fmt.Errorf("cluster zone cannot be empty")
	}

	// Validate parsed VM components
	if c.VMScope == "" {
		return fmt.Errorf("VM scope cannot be empty")
	}
	if c.VMEnv == "" {
		return fmt.Errorf("VM env cannot be empty")
	}
	if c.VMRegion == "" {
		return fmt.Errorf("VM region cannot be empty")
	}
	if c.VMZone == "" {
		return fmt.Errorf("VM zone cannot be empty")
	}
	if c.VMRole == "" {
		return fmt.Errorf("VM role cannot be empty")
	}
	if c.VMSeq == "" {
		return fmt.Errorf("VM sequence cannot be empty")
	}

	return nil
}

// ServerID returns the server identifier in format: {vmName}.{instanceID}
func (c *Cluster) ServerID(instanceID id.ULID) string {
	return fmt.Sprintf("%s.%s", c.VMName, instanceID)
}

// Site returns the site identifier in format: {region}-{zone}
func (c *Cluster) Site() string {
	return fmt.Sprintf("%s-%s", c.ClusterRegion, c.ClusterZone)
}
