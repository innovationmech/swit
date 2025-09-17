package messaging

import (
	"context"
	"testing"
	"time"
)

func TestPlanBrokerSwitchSameType(t *testing.T) {
	current := &BrokerConfig{
		Type:      BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	}
	target := &BrokerConfig{
		Type:      BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	}

	plan, err := PlanBrokerSwitch(context.Background(), current, target)
	if err != nil {
		t.Fatalf("PlanBrokerSwitch returned error: %v", err)
	}

	if plan == nil {
		t.Fatal("expected non-nil switch plan")
	}

	if plan.MigrationDifficulty != MigrationDifficultyLow {
		t.Fatalf("expected low migration difficulty, got %v", plan.MigrationDifficulty)
	}

	if plan.CompatibilityScore != 100 {
		t.Fatalf("expected compatibility score 100, got %d", plan.CompatibilityScore)
	}

	if len(plan.FeatureDeltas) != 0 {
		t.Fatalf("expected no feature deltas for identical brokers, got %d", len(plan.FeatureDeltas))
	}

	if plan.DualWriteRecommended {
		t.Fatal("did not expect dual write recommendation for identical brokers")
	}

	// Ensure originals were not defaulted in-place.
	if current.Connection.Timeout != 0 {
		t.Fatal("expected current config connection timeout to remain default zero value")
	}
}

func TestPlanBrokerSwitchRabbitToKafka(t *testing.T) {
	current := &BrokerConfig{
		Type:      BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://guest:guest@localhost:5672/"},
		Connection: ConnectionConfig{
			Timeout:     5 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    4,
			IdleTimeout: 90 * time.Second,
		},
	}
	target := &BrokerConfig{
		Type:      BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
		Connection: ConnectionConfig{
			Timeout:     5 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    3,
			IdleTimeout: 90 * time.Second,
		},
	}

	plan, err := PlanBrokerSwitch(context.Background(), current, target)
	if err != nil {
		t.Fatalf("PlanBrokerSwitch returned error: %v", err)
	}

	if plan.OverallCompatibility == CompatibilityLevelFull {
		t.Fatal("expected less than full compatibility for RabbitMQ -> Kafka migration")
	}

	if len(plan.FeatureDeltas) == 0 {
		t.Fatal("expected feature deltas for RabbitMQ -> Kafka migration")
	}

	var foundDeadLetterLoss bool
	for _, delta := range plan.FeatureDeltas {
		if delta.Feature == "dead_letter" && delta.Delta == FeatureDeltaTypeLoss {
			foundDeadLetterLoss = true
			if delta.Recommendation == "" {
				t.Fatal("expected a recommendation for dead letter fallback")
			}
		}
	}

	if !foundDeadLetterLoss {
		t.Fatal("expected dead_letter loss when migrating from RabbitMQ to Kafka")
	}

	if !plan.DualWriteRecommended {
		t.Fatal("expected dual write recommendation when losing critical features")
	}

	if len(plan.Checklist) == 0 {
		t.Fatal("expected non-empty checklist")
	}

	if len(plan.Steps) == 0 {
		t.Fatal("expected migration steps from guidance")
	}

	if len(plan.ConfigurationIssues) == 0 {
		t.Fatal("expected configuration issues to guide migration")
	}
}
