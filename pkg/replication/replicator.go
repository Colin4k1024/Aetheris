// Copyright 2026 Aetheris
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package replication

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Replicator 跨区域复制器
type Replicator struct {
	config      *Config
	localStore  Store
	remotePeers []Peer
	mu          sync.RWMutex
	stopChan    chan struct{}
}

// Config 复制配置
type Config struct {
	RegionID        string
	ReplicationMode string // sync, async
	BatchSize       int
	BatchTimeout    time.Duration
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
}

// Peer 远程区域节点
type Peer struct {
	RegionID string
	Endpoint string
	Priority int
	Disabled bool
}

// Store 本地存储接口
type Store interface {
	GetEvents(ctx context.Context, jobID string) ([]Event, error)
	AppendEvent(ctx context.Context, event *Event) error
}

// Event 复制事件
type Event struct {
	JobID     string          `json:"job_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	Version   int64           `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
	RegionID  string          `json:"region_id"`
}

// ReplicationEvent 复制事件消息
type ReplicationEvent struct {
	Event        Event  `json:"event"`
	SourceRegion string `json:"source_region"`
	SequenceNum  int64  `json:"sequence_num"`
}

// NewReplicator 创建新的复制器
func NewReplicator(cfg *Config, localStore Store) (*Replicator, error) {
	r := &Replicator{
		config:      cfg,
		localStore:  localStore,
		remotePeers: make([]Peer, 0),
		stopChan:    make(chan struct{}),
	}

	if cfg.RedisAddr != "" {
		// 初始化 Redis 复制通道
	}

	return r, nil
}

// AddPeer 添加远程节点
func (r *Replicator) AddPeer(peer Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.remotePeers {
		if p.RegionID == peer.RegionID {
			r.remotePeers[i] = peer
			return
		}
	}
	r.remotePeers = append(r.remotePeers, peer)
}

// RemovePeer 移除远程节点
func (r *Replicator) RemovePeer(regionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.remotePeers {
		if p.RegionID == regionID {
			r.remotePeers = append(r.remotePeers[:i], r.remotePeers[i+1:]...)
			return
		}
	}
}

// Start 启动复制器
func (r *Replicator) Start(ctx context.Context) error {
	r.mu.Lock()
	stopChan := r.stopChan
	r.mu.Unlock()

	// 启动事件监听和复制
	go r.listenLocalEvents(ctx, stopChan)

	// 启动从远程节点接收事件
	go r.receiveFromPeers(ctx, stopChan)

	return nil
}

// Stop 停止复制器
func (r *Replicator) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	close(r.stopChan)
}

// listenLocalEvents 监听本地事件并复制到远程
func (r *Replicator) listenLocalEvents(ctx context.Context, stopChan <-chan struct{}) {
	// TODO: 实现本地事件监听
	// 1. 监听本地 job events
	// 2. 批量发送到远程节点
}

// receiveFromPeers 从远程节点接收事件
func (r *Replicator) receiveFromPeers(ctx context.Context, stopChan <-chan struct{}) {
	// TODO: 实现远程事件接收
	// 1. 订阅 Redis 频道
	// 2. 接收并应用远程事件
	// 3. 处理冲突
}

// ReplicateEvent 复制单个事件
func (r *Replicator) ReplicateEvent(ctx context.Context, event *Event) error {
	r.mu.RLock()
	peers := r.remotePeers
	r.mu.RUnlock()

	event.RegionID = r.config.RegionID

	var wg sync.WaitGroup
	errChan := make(chan error, len(peers))

	for _, peer := range peers {
		if peer.Disabled {
			continue
		}

		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()
			if err := r.sendToPeer(ctx, p, event); err != nil {
				errChan <- fmt.Errorf("failed to replicate to %s: %w", p.RegionID, err)
			}
		}(peer)
	}

	wg.Wait()
	close(errChan)

	// 返回第一个错误（如果有）
	for err := range errChan {
		return err
	}

	return nil
}

// sendToPeer 发送到单个远程节点
func (r *Replicator) sendToPeer(ctx context.Context, peer Peer, event *Event) error {
	// TODO: 实现发送到远程节点
	// 可以使用 HTTP/gRPC/Redis PubSub
	return nil
}

// ResolveConflict 解决冲突
// 返回要保留的事件
func (r *Replicator) ResolveConflict(local, remote *Event) (*Event, ConflictResolution) {
	// 基于时间戳和版本号的简单冲突解决策略
	if remote.Version > local.Version {
		return remote, ConflictRemoteWins
	}
	if local.Version > remote.Version {
		return local, ConflictLocalWins
	}

	// 版本相同，按时间戳
	if remote.Timestamp.After(local.Timestamp) {
		return remote, ConflictRemoteWins
	}

	return local, ConflictLocalWins
}

// ConflictResolution 冲突解决策略
type ConflictResolution string

const (
	ConflictLocalWins  ConflictResolution = "local_wins"
	ConflictRemoteWins ConflictResolution = "remote_wins"
	ConflictMerged     ConflictResolution = "merged"
)

// ReplicatorStats 复制统计
type ReplicatorStats struct {
	EventsSent     int64
	EventsReceived int64
	Conflicts      int64
	LastSyncTime   time.Time
	LatencyAvg     time.Duration
}

// GetStats 获取复制统计
func (r *Replicator) GetStats() ReplicatorStats {
	// TODO: 返回实际统计
	return ReplicatorStats{}
}

// RedisReplicator 基于 Redis 的复制实现
type RedisReplicator struct {
	client     *redis.Client
	channel    string
	regionID   string
	localStore Store
}

// NewRedisReplicator 创建 Redis 复制器
func NewRedisReplicator(cfg *Config, store Store) (*RedisReplicator, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisReplicator{
		client:     client,
		channel:    fmt.Sprintf("aetheris:replication:%s", cfg.RegionID),
		regionID:   cfg.RegionID,
		localStore: store,
	}, nil
}

// Publish 发布事件到远程
func (r *RedisReplicator) Publish(ctx context.Context, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return r.client.Publish(ctx, r.channel, data).Err()
}

// Subscribe 订阅远程事件
func (r *RedisReplicator) Subscribe(ctx context.Context) (<-chan *Event, error) {
	pubsub := r.client.Subscribe(ctx, r.channel)

	ch := make(chan *Event, 100)
	go func() {
		for msg := range pubsub.Channel() {
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}
			ch <- &event
		}
		close(ch)
	}()

	return ch, nil
}

// Close 关闭连接
func (r *RedisReplicator) Close() error {
	return r.client.Close()
}
