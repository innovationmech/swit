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

package messaging

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateCapabilitiesMarkdown renders a cross-broker feature parity matrix in Markdown.
// The output is intended to be committed under docs (e.g. docs/pages/en/reference/capabilities.md).
// This function focuses on the three production brokers (Kafka, RabbitMQ, NATS).
func GenerateCapabilitiesMarkdown() (string, error) {
	type brokerEntry struct {
		name       string
		brokerType BrokerType
		caps       *BrokerCapabilities
	}

	brokers := []brokerEntry{
		{name: "Kafka", brokerType: BrokerTypeKafka},
		{name: "RabbitMQ", brokerType: BrokerTypeRabbitMQ},
		{name: "NATS", brokerType: BrokerTypeNATS},
	}

	for i := range brokers {
		c, err := GetCapabilityProfile(brokers[i].brokerType)
		if err != nil {
			return "", fmt.Errorf("get capabilities for %s: %w", brokers[i].brokerType, err)
		}
		brokers[i].caps = c
	}

	// Header
	var b strings.Builder
	b.WriteString("# Message Broker Capabilities Matrix\n\n")
	b.WriteString("This document summarizes supported capabilities across major brokers and suggests graceful degradation strategies.\n\n")

	// Core feature matrix
	b.WriteString("## Core Features\n\n")
	b.WriteString("| Feature | Kafka | RabbitMQ | NATS |\n")
	b.WriteString("|---|---:|---:|---:|\n")

	// Helper to format boolean as check/cross
	check := func(v bool) string {
		if v {
			return "✅"
		}
		return "❌"
	}

	row := func(name string, vals ...bool) {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			name, check(vals[0]), check(vals[1]), check(vals[2]),
		))
	}

	row("Transactions", brokers[0].caps.SupportsTransactions, brokers[1].caps.SupportsTransactions, brokers[2].caps.SupportsTransactions)
	row("Ordering", brokers[0].caps.SupportsOrdering, brokers[1].caps.SupportsOrdering, brokers[2].caps.SupportsOrdering)
	row("Partitioning", brokers[0].caps.SupportsPartitioning, brokers[1].caps.SupportsPartitioning, brokers[2].caps.SupportsPartitioning)
	row("Dead Letter", brokers[0].caps.SupportsDeadLetter, brokers[1].caps.SupportsDeadLetter, brokers[2].caps.SupportsDeadLetter)
	row("Delayed Delivery", brokers[0].caps.SupportsDelayedDelivery, brokers[1].caps.SupportsDelayedDelivery, brokers[2].caps.SupportsDelayedDelivery)
	row("Priority", brokers[0].caps.SupportsPriority, brokers[1].caps.SupportsPriority, brokers[2].caps.SupportsPriority)
	row("Streaming", brokers[0].caps.SupportsStreaming, brokers[1].caps.SupportsStreaming, brokers[2].caps.SupportsStreaming)
	row("Seek / Replay", brokers[0].caps.SupportsSeek, brokers[1].caps.SupportsSeek, brokers[2].caps.SupportsSeek)
	row("Consumer Groups", brokers[0].caps.SupportsConsumerGroups, brokers[1].caps.SupportsConsumerGroups, brokers[2].caps.SupportsConsumerGroups)

	b.WriteString("\n")

	// Limits & formats
	b.WriteString("## Limits & Formats\n\n")
	b.WriteString("| Property | Kafka | RabbitMQ | NATS |\n")
	b.WriteString("|---|---:|---:|---:|\n")
	b.WriteString(fmt.Sprintf("| Max Message Size | %d | %d | %d |\n",
		brokers[0].caps.MaxMessageSize, brokers[1].caps.MaxMessageSize, brokers[2].caps.MaxMessageSize,
	))
	b.WriteString(fmt.Sprintf("| Max Batch Size | %d | %d | %d |\n",
		brokers[0].caps.MaxBatchSize, brokers[1].caps.MaxBatchSize, brokers[2].caps.MaxBatchSize,
	))

	// Compression & serialization
	toCSVCompression := func(vals []CompressionType) string {
		if len(vals) == 0 {
			return "-"
		}
		ss := make([]string, 0, len(vals))
		for _, v := range vals {
			ss = append(ss, string(v))
		}
		sort.Strings(ss)
		return strings.Join(ss, ", ")
	}
	toCSVSerialization := func(vals []SerializationType) string {
		if len(vals) == 0 {
			return "-"
		}
		ss := make([]string, 0, len(vals))
		for _, v := range vals {
			ss = append(ss, string(v))
		}
		sort.Strings(ss)
		return strings.Join(ss, ", ")
	}
	b.WriteString(fmt.Sprintf("| Compression | %s | %s | %s |\n",
		toCSVCompression(brokers[0].caps.SupportedCompressionTypes),
		toCSVCompression(brokers[1].caps.SupportedCompressionTypes),
		toCSVCompression(brokers[2].caps.SupportedCompressionTypes),
	))
	b.WriteString(fmt.Sprintf("| Serialization | %s | %s | %s |\n",
		toCSVSerialization(brokers[0].caps.SupportedSerializationTypes),
		toCSVSerialization(brokers[1].caps.SupportedSerializationTypes),
		toCSVSerialization(brokers[2].caps.SupportedSerializationTypes),
	))

	// Graceful degradation notes
	b.WriteString("\n## Graceful Degradation\n\n")
	b.WriteString("- Transactions: Use transactional outbox + idempotent consumers when unsupported.\n")
	b.WriteString("- Ordering: Partition by key or design idempotent handlers to tolerate reordering.\n")
	b.WriteString("- Partitioning: Emulate via routing keys/subjects; shard by naming convention.\n")
	b.WriteString("- Dead Letter: Publish to error/parking topics and process with DLQ worker if no native DLQ.\n")
	b.WriteString("- Delayed Delivery: Use delayed queues (RabbitMQ) or schedule via consumer timers/cron.\n")
	b.WriteString("- Priority: Separate topics/queues by priority and weight consumers accordingly.\n")
	b.WriteString("- Seek/Replay: Persist offsets and re-consume from stored positions when broker lacks seek.\n")

	// Extended notes
	b.WriteString("\n## Extended Capabilities\n\n")
	b.WriteString("Some adapters expose extended keys in the capabilities map (e.g., `nats.jetstream`, `nats.key_value_store`, `rabbitmq.exchanges`).\n")

	return b.String(), nil
}
