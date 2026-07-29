package cluster

import (
	"fmt"
	"regexp"
)

// clusterNameRegex validates cluster name format: scope-env-region+zone
// Example: core-prod-eu-central-1a (region: eu-central-1, zone: a)
var clusterNameRegex = regexp.MustCompile(`^([a-z0-9]+)-([a-z0-9]+)-([a-z0-9]+-[a-z0-9]+-[0-9]+[a-z])$`)

// vmNameRegex validates VM name format: scope-env-region+zone-role-seq
// Example: core-p-eu1a-all-01 (region: eu1, zone: a)
var vmNameRegex = regexp.MustCompile(`^([a-z0-9]+)-([a-z0-9]+)-([a-z0-9]+[a-z])-([a-z0-9]+)-([0-9]+)$`)

// ClusterComponents holds parsed cluster name components
type ClusterComponents struct {
	Scope  string
	Env    string
	Region string
	Zone   string
}

// VMComponents holds parsed VM name components
type VMComponents struct {
	Scope  string
	Env    string
	Region string
	Zone   string
	Role   string
	Seq    string
}

// ParseClusterName parses a cluster name in format: scope-env-region+zone
// Example: "core-prod-eu-central-1a" -> scope="core", env="prod", region="eu-central-1", zone="a"
func ParseClusterName(name string) (*ClusterComponents, error) {
	if name == "" {
		return nil, fmt.Errorf("cluster name cannot be empty")
	}

	matches := clusterNameRegex.FindStringSubmatch(name)
	if matches == nil {
		return nil, fmt.Errorf("invalid cluster name format: %s (expected format: scope-env-region+zone, e.g., core-prod-eu-central-1a)", name)
	}

	// Extract region+zone and split to get region and zone
	regionZone := matches[3]
	// Zone is the last character
	zone := regionZone[len(regionZone)-1:]
	region := regionZone[:len(regionZone)-1]

	return &ClusterComponents{
		Scope:  matches[1],
		Env:    matches[2],
		Region: region,
		Zone:   zone,
	}, nil
}

// BuildClusterName constructs a cluster name from components
// Format: scope-env-region+zone (no dash between region and zone)
func BuildClusterName(scope, env, region, zone string) string {
	return fmt.Sprintf("%s-%s-%s%s", scope, env, region, zone)
}

// ParseVMName parses a VM name in format: scope-env-region+zone-role-seq
// Example: "core-p-eu1a-all-01" -> scope="core", env="p", region="eu1", zone="a", role="all", seq="01"
// Note: VM region is simplified format (e.g., "eu1" not "eu-central-1")
func ParseVMName(name string) (*VMComponents, error) {
	if name == "" {
		return nil, fmt.Errorf("VM name cannot be empty")
	}

	matches := vmNameRegex.FindStringSubmatch(name)
	if matches == nil {
		return nil, fmt.Errorf("invalid VM name format: %s (expected format: scope-env-region+zone-role-seq, e.g., core-p-eu1a-all-01)", name)
	}

	// Extract region+zone and split to get region and zone
	regionZone := matches[3]
	// Zone is the last character
	zone := regionZone[len(regionZone)-1:]
	region := regionZone[:len(regionZone)-1]

	return &VMComponents{
		Scope:  matches[1],
		Env:    matches[2],
		Region: region,
		Zone:   zone,
		Role:   matches[4],
		Seq:    matches[5],
	}, nil
}

// BuildVMName constructs a VM name from components
// Format: scope-env-region+zone-role-seq (no dash between region and zone)
func BuildVMName(scope, env, region, zone, role, seq string) string {
	return fmt.Sprintf("%s-%s-%s%s-%s-%s", scope, env, region, zone, role, seq)
}
