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
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewClusterDiscovery(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000", "localhost:7001", "localhost:7002"},
	}
	config.ApplyDefaults()

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, err := NewClusterDiscovery(clusterClient, config)
	if err != nil {
		t.Fatalf("Failed to create cluster discovery: %v", err)
	}

	if discovery == nil {
		t.Fatal("Expected discovery to be created")
	}

	if discovery.client != clusterClient {
		t.Error("Expected discovery client to match")
	}

	if discovery.config != config {
		t.Error("Expected discovery config to match")
	}

	if !discovery.discoveryEnabled {
		t.Error("Expected discovery to be enabled")
	}
}

func TestNewClusterDiscovery_NotClusterMode(t *testing.T) {
	config := &RedisConfig{
		Mode: RedisModeStandalone,
		Addr: "localhost:6379",
	}
	config.ApplyDefaults()

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	_, err := NewClusterDiscovery(client, config)
	if err != ErrNotClusterMode {
		t.Errorf("Expected ErrNotClusterMode, got %v", err)
	}
}

func TestNewClusterDiscovery_WrongClientType(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}
	config.ApplyDefaults()

	// Create standalone client even though config says cluster
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	_, err := NewClusterDiscovery(client, config)
	if err != ErrNotClusterMode {
		t.Errorf("Expected ErrNotClusterMode for wrong client type, got %v", err)
	}
}

func TestParseClusterInfo(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	tests := []struct {
		name             string
		input            string
		expectedState    string
		expectedAssigned int
		expectedOk       int
		expectedError    int
	}{
		{
			name: "healthy cluster",
			input: `cluster_state:ok
cluster_slots_assigned:16384
cluster_slots_ok:16384
cluster_slots_pfail:0
cluster_slots_fail:0`,
			expectedState:    "ok",
			expectedAssigned: 16384,
			expectedOk:       16384,
			expectedError:    0,
		},
		{
			name: "cluster with failures",
			input: `cluster_state:fail
cluster_slots_assigned:16384
cluster_slots_ok:10000
cluster_slots_pfail:100
cluster_slots_fail:200`,
			expectedState:    "fail",
			expectedAssigned: 16384,
			expectedOk:       10000,
			expectedError:    300,
		},
		{
			name: "partial assignment",
			input: `cluster_state:ok
cluster_slots_assigned:8192
cluster_slots_ok:8192
cluster_slots_pfail:0
cluster_slots_fail:0`,
			expectedState:    "ok",
			expectedAssigned: 8192,
			expectedOk:       8192,
			expectedError:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, assigned, ok, errorCount := discovery.parseClusterInfo(tt.input)

			if state != tt.expectedState {
				t.Errorf("Expected state %s, got %s", tt.expectedState, state)
			}

			if assigned != tt.expectedAssigned {
				t.Errorf("Expected assigned %d, got %d", tt.expectedAssigned, assigned)
			}

			if ok != tt.expectedOk {
				t.Errorf("Expected ok %d, got %d", tt.expectedOk, ok)
			}

			if errorCount != tt.expectedError {
				t.Errorf("Expected error %d, got %d", tt.expectedError, errorCount)
			}
		})
	}
}

func TestParseNodeLine(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	tests := []struct {
		name         string
		input        string
		expectedRole string
		expectedAddr string
		wantErr      bool
	}{
		{
			name:         "master node",
			input:        "07c37dfeb235213a872192d90877d0cd55635b91 127.0.0.1:7000@17000 master - 0 1426238316232 2 connected 5461-10922",
			expectedRole: "master",
			expectedAddr: "127.0.0.1:7000",
			wantErr:      false,
		},
		{
			name:         "slave node",
			input:        "67ed2db8d677e59ec4a4cefb06858cf2a1a89fa1 127.0.0.1:7002@17002 slave 07c37dfeb235213a872192d90877d0cd55635b91 0 1426238317741 3 connected",
			expectedRole: "slave",
			expectedAddr: "127.0.0.1:7002",
			wantErr:      false,
		},
		{
			name:    "invalid line",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := discovery.parseNodeLine(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if node.Role != tt.expectedRole {
				t.Errorf("Expected role %s, got %s", tt.expectedRole, node.Role)
			}

			if node.Addr != tt.expectedAddr {
				t.Errorf("Expected addr %s, got %s", tt.expectedAddr, node.Addr)
			}
		})
	}
}

func TestParseNodeLine_SlotRanges(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	// Master with slot range
	input := "07c37dfeb235213a872192d90877d0cd55635b91 127.0.0.1:7000@17000 master - 0 1426238316232 2 connected 0-5460 5461-10922"
	node, err := discovery.parseNodeLine(input)
	if err != nil {
		t.Fatalf("Failed to parse node line: %v", err)
	}

	if len(node.SlotRanges) != 2 {
		t.Errorf("Expected 2 slot ranges, got %d", len(node.SlotRanges))
	}

	if node.SlotRanges[0].Start != 0 || node.SlotRanges[0].End != 5460 {
		t.Errorf("Expected slot range 0-5460, got %d-%d", node.SlotRanges[0].Start, node.SlotRanges[0].End)
	}

	if node.SlotRanges[1].Start != 5461 || node.SlotRanges[1].End != 10922 {
		t.Errorf("Expected slot range 5461-10922, got %d-%d", node.SlotRanges[1].Start, node.SlotRanges[1].End)
	}
}

func TestCopyTopology(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	// Create original topology
	original := &ClusterTopology{
		Nodes: []*ClusterNodeInfo{
			{
				ID:    "node1",
				Addr:  "127.0.0.1:7000",
				Role:  "master",
				Flags: []string{"master"},
				SlotRanges: []SlotRange{
					{Start: 0, End: 5460},
				},
			},
		},
		Masters: []*ClusterNodeInfo{},
		Slaves:  []*ClusterNodeInfo{},
		SlotDistribution: map[string][]SlotRange{
			"node1": {{Start: 0, End: 5460}},
		},
		Timestamp:            time.Now(),
		ClusterState:         "ok",
		ClusterSlotsAssigned: 16384,
	}

	// Make a copy
	copied := discovery.copyTopology(original)

	if copied == nil {
		t.Fatal("Expected copied topology to be non-nil")
	}

	// Verify it's a different instance
	if copied == original {
		t.Error("Expected a copy, not the same instance")
	}

	// Verify values match
	if copied.ClusterState != original.ClusterState {
		t.Error("Expected cluster state to match")
	}

	if len(copied.Nodes) != len(original.Nodes) {
		t.Error("Expected same number of nodes")
	}

	// Modify original and verify copy is unchanged
	original.ClusterState = "fail"
	if copied.ClusterState == "fail" {
		t.Error("Copy was affected by modification to original")
	}
}

func TestCopyTopology_Nil(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	copied := discovery.copyTopology(nil)
	if copied != nil {
		t.Error("Expected nil when copying nil topology")
	}
}

func TestGetLastTopology(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	// Should return nil initially
	topology := discovery.GetLastTopology()
	if topology != nil {
		t.Error("Expected nil topology initially")
	}

	// Set a topology
	discovery.lastTopology = &ClusterTopology{
		ClusterState: "ok",
		Nodes:        []*ClusterNodeInfo{},
	}

	// Should return a copy
	topology = discovery.GetLastTopology()
	if topology == nil {
		t.Error("Expected topology to be returned")
	}

	if topology == discovery.lastTopology {
		t.Error("Expected a copy, not the same instance")
	}
}

func TestStartDiscovery(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		discovery.StartDiscovery(ctx, 50*time.Millisecond)
		close(done)
	}()

	// Wait for context to be cancelled
	<-done

	// Function should have returned when context was cancelled
}

func TestStartDiscovery_Disabled(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)
	discovery.discoveryEnabled = false

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		discovery.StartDiscovery(ctx, 50*time.Millisecond)
		close(done)
	}()

	// Should return immediately
	select {
	case <-done:
		// Expected
	case <-time.After(150 * time.Millisecond):
		t.Error("StartDiscovery should return immediately when disabled")
	}
}

func TestGetMasterForSlot(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	// Set up test topology
	discovery.lastTopology = &ClusterTopology{
		Masters: []*ClusterNodeInfo{
			{
				ID:   "master1",
				Addr: "127.0.0.1:7000",
				Role: "master",
			},
			{
				ID:   "master2",
				Addr: "127.0.0.1:7001",
				Role: "master",
			},
		},
		SlotDistribution: map[string][]SlotRange{
			"master1": {{Start: 0, End: 5460}},
			"master2": {{Start: 5461, End: 10922}},
		},
	}

	tests := []struct {
		name         string
		slot         int
		expectedID   string
		expectedAddr string
		wantErr      bool
	}{
		{
			name:         "slot in first master range",
			slot:         100,
			expectedID:   "master1",
			expectedAddr: "127.0.0.1:7000",
			wantErr:      false,
		},
		{
			name:         "slot in second master range",
			slot:         6000,
			expectedID:   "master2",
			expectedAddr: "127.0.0.1:7001",
			wantErr:      false,
		},
		{
			name:    "slot not assigned",
			slot:    15000,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := discovery.GetMasterForSlot(tt.slot)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if node.ID != tt.expectedID {
				t.Errorf("Expected node ID %s, got %s", tt.expectedID, node.ID)
			}

			if node.Addr != tt.expectedAddr {
				t.Errorf("Expected node addr %s, got %s", tt.expectedAddr, node.Addr)
			}
		})
	}
}

func TestGetMasterForSlot_NoTopology(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	_, err := discovery.GetMasterForSlot(100)
	if err == nil {
		t.Error("Expected error when no topology available")
	}
}

func TestGetSlavesForMaster(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	// Set up test topology
	discovery.lastTopology = &ClusterTopology{
		Slaves: []*ClusterNodeInfo{
			{
				ID:       "slave1",
				Addr:     "127.0.0.1:7003",
				Role:     "slave",
				MasterID: "master1",
			},
			{
				ID:       "slave2",
				Addr:     "127.0.0.1:7004",
				Role:     "slave",
				MasterID: "master1",
			},
			{
				ID:       "slave3",
				Addr:     "127.0.0.1:7005",
				Role:     "slave",
				MasterID: "master2",
			},
		},
	}

	tests := []struct {
		name          string
		masterID      string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "master with two slaves",
			masterID:      "master1",
			expectedCount: 2,
			expectedIDs:   []string{"slave1", "slave2"},
		},
		{
			name:          "master with one slave",
			masterID:      "master2",
			expectedCount: 1,
			expectedIDs:   []string{"slave3"},
		},
		{
			name:          "master with no slaves",
			masterID:      "master3",
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slaves := discovery.GetSlavesForMaster(tt.masterID)

			if len(slaves) != tt.expectedCount {
				t.Errorf("Expected %d slaves, got %d", tt.expectedCount, len(slaves))
			}

			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, slave := range slaves {
					if slave.ID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected slave %s not found", expectedID)
				}
			}
		})
	}
}

func TestGetSlavesForMaster_NoTopology(t *testing.T) {
	config := &RedisConfig{
		Mode:  RedisModeCluster,
		Addrs: []string{"localhost:7000"},
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.Addrs,
	})
	defer clusterClient.Close()

	discovery, _ := NewClusterDiscovery(clusterClient, config)

	slaves := discovery.GetSlavesForMaster("master1")
	if slaves != nil {
		t.Error("Expected nil when no topology available")
	}
}
