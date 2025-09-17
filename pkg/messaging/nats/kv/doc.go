// Package natskv 提供对 NATS JetStream Key-Value 的轻量封装，
// 便于在 SWIT 中实现分布式配置/状态存储：
// - EnsureBucket：按配置创建或获取 KV 桶
// - OpenBucket：打开已有 KV 桶
// - Put/Create/Get：基本读写接口
// - Watch：键或通配符的变更订阅
//
// 注意：此封装依赖 github.com/nats-io/nats.go 的 KV/JS 抽象，
// 不负责连接管理，调用方应从 NATS broker 或已建立的 JetStreamContext 注入。
package natskv
