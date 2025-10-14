// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrNotClusterMode indicates that the operation requires cluster mode.
	ErrNotClusterMode = errors.New("operation requires Redis cluster mode")

	// ErrNoClusterNodes indicates that no cluster nodes were found.
	ErrNoClusterNodes = errors.New("no cluster nodes found")
)

// ClusterNodeInfo represents information about a Redis cluster node.
type ClusterNodeInfo struct {
	// ID is the unique identifier of the node.
	ID string `json:"id"`

	// Addr is the address of the node (host:port).
	Addr string `json:"addr"`

	// Role is the role of the node (master or slave).
	Role string `json:"role"`

	// MasterID is the ID of the master node (for slaves only).
	MasterID string `json:"master_id,omitempty"`

	// Flags contains the node flags (e.g., master, slave, fail, handshake).
	Flags []string `json:"flags"`

	// SlotRanges contains the hash slot ranges handled by this node.
	SlotRanges []SlotRange `json:"slot_ranges,omitempty"`

	// LinkState is the state of the link to this node.
	LinkState string `json:"link_state"`

	// LastPingSent is when the last PING was sent to this node.
	LastPingSent int64 `json:"last_ping_sent"`

	// LastPongReceived is when the last PONG was received from this node.
	LastPongReceived int64 `json:"last_pong_received"`
}

// SlotRange represents a range of hash slots.
type SlotRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// ClusterTopology represents the topology of a Redis cluster.
type ClusterTopology struct {
	// Nodes contains information about all cluster nodes.
	Nodes []*ClusterNodeInfo `json:"nodes"`

	// Masters contains the list of master nodes.
	Masters []*ClusterNodeInfo `json:"masters"`

	// Slaves contains the list of slave nodes.
	Slaves []*ClusterNodeInfo `json:"slaves"`

	// SlotDistribution maps slot ranges to node IDs.
	SlotDistribution map[string][]SlotRange `json:"slot_distribution"`

	// Timestamp is when this topology was discovered.
	Timestamp time.Time `json:"timestamp"`

	// ClusterState is the overall state of the cluster (ok or fail).
	ClusterState string `json:"cluster_state"`

	// ClusterSlotsAssigned is the number of hash slots assigned.
	ClusterSlotsAssigned int `json:"cluster_slots_assigned"`

	// ClusterSlotsOk is the number of hash slots in OK state.
	ClusterSlotsOk int `json:"cluster_slots_ok"`

	// ClusterSlotsError is the number of hash slots in error state.
	ClusterSlotsError int `json:"cluster_slots_error"`
}

// ClusterDiscovery manages Redis cluster topology discovery and monitoring.
type ClusterDiscovery struct {
	client *redis.ClusterClient
	config *RedisConfig

	// mu protects the topology
	mu sync.RWMutex

	// lastTopology stores the last discovered topology
	lastTopology *ClusterTopology

	// discoveryEnabled indicates if discovery is enabled
	discoveryEnabled bool
}

// NewClusterDiscovery creates a new cluster discovery instance.
func NewClusterDiscovery(client RedisClient, config *RedisConfig) (*ClusterDiscovery, error) {
	if config.Mode != RedisModeCluster {
		return nil, ErrNotClusterMode
	}

	clusterClient, ok := client.(*redis.ClusterClient)
	if !ok {
		return nil, ErrNotClusterMode
	}

	return &ClusterDiscovery{
		client:           clusterClient,
		config:           config,
		discoveryEnabled: true,
	}, nil
}

// DiscoverTopology discovers the current cluster topology.
func (d *ClusterDiscovery) DiscoverTopology(ctx context.Context) (*ClusterTopology, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Get cluster info
	clusterInfo, err := d.client.ClusterInfo(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	// Parse cluster info
	clusterState, slotsAssigned, slotsOk, slotsError := d.parseClusterInfo(clusterInfo)

	// Get cluster nodes
	nodesInfo, err := d.client.ClusterNodes(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	// Parse nodes information
	nodes, err := d.parseClusterNodes(nodesInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cluster nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, ErrNoClusterNodes
	}

	// Categorize nodes
	var masters, slaves []*ClusterNodeInfo
	slotDistribution := make(map[string][]SlotRange)

	for _, node := range nodes {
		if node.Role == "master" {
			masters = append(masters, node)
			if len(node.SlotRanges) > 0 {
				slotDistribution[node.ID] = node.SlotRanges
			}
		} else if node.Role == "slave" {
			slaves = append(slaves, node)
		}
	}

	topology := &ClusterTopology{
		Nodes:                nodes,
		Masters:              masters,
		Slaves:               slaves,
		SlotDistribution:     slotDistribution,
		Timestamp:            time.Now(),
		ClusterState:         clusterState,
		ClusterSlotsAssigned: slotsAssigned,
		ClusterSlotsOk:       slotsOk,
		ClusterSlotsError:    slotsError,
	}

	// Store the topology
	d.mu.Lock()
	d.lastTopology = topology
	d.mu.Unlock()

	return topology, nil
}

// GetLastTopology returns the last discovered cluster topology.
func (d *ClusterDiscovery) GetLastTopology() *ClusterTopology {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.lastTopology == nil {
		return nil
	}

	// Return a deep copy to prevent external modification
	return d.copyTopology(d.lastTopology)
}

// CheckClusterHealth checks the health of the Redis cluster.
func (d *ClusterDiscovery) CheckClusterHealth(ctx context.Context) (bool, []string, error) {
	topology, err := d.DiscoverTopology(ctx)
	if err != nil {
		return false, nil, err
	}

	var issues []string
	healthy := true

	// Check cluster state
	if topology.ClusterState != "ok" {
		issues = append(issues, fmt.Sprintf("Cluster state is %s (expected: ok)", topology.ClusterState))
		healthy = false
	}

	// Check slot coverage
	if topology.ClusterSlotsAssigned != 16384 {
		issues = append(issues, fmt.Sprintf("Not all slots assigned: %d/16384", topology.ClusterSlotsAssigned))
		healthy = false
	}

	if topology.ClusterSlotsError > 0 {
		issues = append(issues, fmt.Sprintf("Slots in error state: %d", topology.ClusterSlotsError))
		healthy = false
	}

	// Check for failed nodes
	for _, node := range topology.Nodes {
		for _, flag := range node.Flags {
			if flag == "fail" || flag == "pfail" {
				issues = append(issues, fmt.Sprintf("Node %s (%s) has flag: %s", node.ID, node.Addr, flag))
				healthy = false
			}
		}

		// Check link state
		if node.LinkState != "connected" {
			issues = append(issues, fmt.Sprintf("Node %s (%s) link state: %s", node.ID, node.Addr, node.LinkState))
		}
	}

	// Check master-slave ratio
	if len(topology.Masters) == 0 {
		issues = append(issues, "No master nodes found")
		healthy = false
	}

	// Warn if no slaves (no replication)
	if len(topology.Slaves) == 0 {
		issues = append(issues, "No slave nodes found (no replication)")
	}

	return healthy, issues, nil
}

// StartDiscovery starts periodic cluster topology discovery.
func (d *ClusterDiscovery) StartDiscovery(ctx context.Context, interval time.Duration) {
	if !d.discoveryEnabled {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Discover topology (ignore errors during monitoring)
			_, _ = d.DiscoverTopology(ctx)
		}
	}
}

// parseClusterInfo parses the CLUSTER INFO output.
func (d *ClusterDiscovery) parseClusterInfo(info string) (state string, assigned, ok, error int) {
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "cluster_state":
			state = value
		case "cluster_slots_assigned":
			fmt.Sscanf(value, "%d", &assigned)
		case "cluster_slots_ok":
			fmt.Sscanf(value, "%d", &ok)
		case "cluster_slots_pfail":
			var pfail int
			fmt.Sscanf(value, "%d", &pfail)
			error += pfail
		case "cluster_slots_fail":
			var fail int
			fmt.Sscanf(value, "%d", &fail)
			error += fail
		}
	}

	return
}

// parseClusterNodes parses the CLUSTER NODES output.
func (d *ClusterDiscovery) parseClusterNodes(nodesInfo string) ([]*ClusterNodeInfo, error) {
	var nodes []*ClusterNodeInfo

	lines := strings.Split(nodesInfo, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		node, err := d.parseNodeLine(line)
		if err != nil {
			// Skip malformed lines
			continue
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// parseNodeLine parses a single line from CLUSTER NODES output.
func (d *ClusterDiscovery) parseNodeLine(line string) (*ClusterNodeInfo, error) {
	// Format: <id> <ip:port@cport> <flags> <master> <ping-sent> <pong-recv> <config-epoch> <link-state> <slot> <slot> ... <slot>
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return nil, fmt.Errorf("invalid node line: %s", line)
	}

	node := &ClusterNodeInfo{
		ID:        fields[0],
		Addr:      strings.Split(fields[1], "@")[0], // Remove cluster bus port
		Flags:     strings.Split(fields[2], ","),
		MasterID:  fields[3],
		LinkState: fields[7],
	}

	// Parse ping/pong times
	fmt.Sscanf(fields[4], "%d", &node.LastPingSent)
	fmt.Sscanf(fields[5], "%d", &node.LastPongReceived)

	// Determine role
	for _, flag := range node.Flags {
		if flag == "master" {
			node.Role = "master"
		} else if flag == "slave" {
			node.Role = "slave"
		}
	}

	// For masters, parse slot ranges
	if node.Role == "master" && len(fields) > 8 {
		for _, slotInfo := range fields[8:] {
			// Slot info can be: single slot, range (start-end), or importing/migrating
			if strings.HasPrefix(slotInfo, "[") {
				// Skip importing/migrating slots
				continue
			}

			var slotRange SlotRange
			if strings.Contains(slotInfo, "-") {
				// Slot range
				parts := strings.Split(slotInfo, "-")
				if len(parts) == 2 {
					fmt.Sscanf(parts[0], "%d", &slotRange.Start)
					fmt.Sscanf(parts[1], "%d", &slotRange.End)
					node.SlotRanges = append(node.SlotRanges, slotRange)
				}
			} else {
				// Single slot
				var slot int
				if _, err := fmt.Sscanf(slotInfo, "%d", &slot); err == nil {
					slotRange.Start = slot
					slotRange.End = slot
					node.SlotRanges = append(node.SlotRanges, slotRange)
				}
			}
		}
	}

	return node, nil
}

// copyTopology creates a deep copy of the topology.
func (d *ClusterDiscovery) copyTopology(topology *ClusterTopology) *ClusterTopology {
	if topology == nil {
		return nil
	}

	copy := &ClusterTopology{
		Nodes:                make([]*ClusterNodeInfo, len(topology.Nodes)),
		Masters:              make([]*ClusterNodeInfo, len(topology.Masters)),
		Slaves:               make([]*ClusterNodeInfo, len(topology.Slaves)),
		SlotDistribution:     make(map[string][]SlotRange),
		Timestamp:            topology.Timestamp,
		ClusterState:         topology.ClusterState,
		ClusterSlotsAssigned: topology.ClusterSlotsAssigned,
		ClusterSlotsOk:       topology.ClusterSlotsOk,
		ClusterSlotsError:    topology.ClusterSlotsError,
	}

	// Deep copy nodes
	for i, node := range topology.Nodes {
		copy.Nodes[i] = d.copyNodeInfo(node)
	}

	for i, node := range topology.Masters {
		copy.Masters[i] = d.copyNodeInfo(node)
	}

	for i, node := range topology.Slaves {
		copy.Slaves[i] = d.copyNodeInfo(node)
	}

	// Copy slot distribution
	for nodeID, ranges := range topology.SlotDistribution {
		copy.SlotDistribution[nodeID] = make([]SlotRange, len(ranges))
		for i, r := range ranges {
			copy.SlotDistribution[nodeID][i] = r
		}
	}

	return copy
}

// copyNodeInfo creates a copy of a node info.
func (d *ClusterDiscovery) copyNodeInfo(node *ClusterNodeInfo) *ClusterNodeInfo {
	if node == nil {
		return nil
	}

	copy := &ClusterNodeInfo{
		ID:               node.ID,
		Addr:             node.Addr,
		Role:             node.Role,
		MasterID:         node.MasterID,
		Flags:            make([]string, len(node.Flags)),
		SlotRanges:       make([]SlotRange, len(node.SlotRanges)),
		LinkState:        node.LinkState,
		LastPingSent:     node.LastPingSent,
		LastPongReceived: node.LastPongReceived,
	}

	for i, flag := range node.Flags {
		copy.Flags[i] = flag
	}

	for i, slotRange := range node.SlotRanges {
		copy.SlotRanges[i] = slotRange
	}

	return copy
}

// GetMasterForSlot returns the master node that handles the given slot.
func (d *ClusterDiscovery) GetMasterForSlot(slot int) (*ClusterNodeInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.lastTopology == nil {
		return nil, fmt.Errorf("no topology available")
	}

	for nodeID, ranges := range d.lastTopology.SlotDistribution {
		for _, r := range ranges {
			if slot >= r.Start && slot <= r.End {
				// Find the node by ID
				for _, node := range d.lastTopology.Masters {
					if node.ID == nodeID {
						return d.copyNodeInfo(node), nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no master found for slot %d", slot)
}

// GetSlavesForMaster returns all slave nodes for a given master.
func (d *ClusterDiscovery) GetSlavesForMaster(masterID string) []*ClusterNodeInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.lastTopology == nil {
		return nil
	}

	var slaves []*ClusterNodeInfo
	for _, node := range d.lastTopology.Slaves {
		if node.MasterID == masterID {
			slaves = append(slaves, d.copyNodeInfo(node))
		}
	}

	return slaves
}

