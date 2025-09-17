package nats

import "github.com/nats-io/nats.go"

// JetStreamProvider 由支持 JetStream 的 broker 实现，用于暴露 JS 上下文。
type JetStreamProvider interface {
	GetJetStream() nats.JetStreamContext
}

// GetJetStreamFrom 尝试从通用 broker 获取 JetStreamContext。
func GetJetStreamFrom(broker interface{}) (nats.JetStreamContext, bool) {
	if p, ok := broker.(JetStreamProvider); ok {
		return p.GetJetStream(), true
	}
	return nil, false
}
