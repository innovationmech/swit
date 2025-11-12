// Copyright (c) 2024 Six-Thirty Labs, Inc.
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

package opa

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher 策略文件监听器
type Watcher struct {
	manager   *Manager
	fsWatcher *fsnotify.Watcher
	watchDir  string
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	running   bool
}

// NewWatcher 创建文件监听器
func NewWatcher(manager *Manager, ctx context.Context) (*Watcher, error) {
	if manager == nil {
		return nil, fmt.Errorf("manager cannot be nil")
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	watcherCtx, cancel := context.WithCancel(ctx)

	return &Watcher{
		manager:   manager,
		fsWatcher: fsWatcher,
		ctx:       watcherCtx,
		cancel:    cancel,
	}, nil
}

// Start 启动监听器
func (w *Watcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher is already running")
	}
	w.running = true
	w.mu.Unlock()

	// 获取策略目录
	config := w.manager.client.(*embeddedClient).config
	if config.EmbeddedConfig == nil || config.EmbeddedConfig.PolicyDir == "" {
		return fmt.Errorf("policy directory not configured")
	}

	w.watchDir = config.EmbeddedConfig.PolicyDir

	// 添加监听目录
	if err := w.fsWatcher.Add(w.watchDir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", w.watchDir, err)
	}

	// 启动监听协程
	go w.watch()

	return nil
}

// Stop 停止监听器
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	w.cancel()
	w.running = false

	return w.fsWatcher.Close()
}

// watch 监听文件变化
func (w *Watcher) watch() {
	// 防抖动：在短时间内只处理一次文件变更
	debounceMap := make(map[string]time.Time)
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// 只处理 .rego 文件
			if filepath.Ext(event.Name) != ".rego" {
				continue
			}

			// 防抖动检查
			lastTime, exists := debounceMap[event.Name]
			if exists && time.Since(lastTime) < debounceDuration {
				continue
			}
			debounceMap[event.Name] = time.Now()

			// 处理文件事件
			if err := w.handleFileEvent(event); err != nil {
				log.Printf("Failed to handle file event %s: %v", event.Name, err)
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)

		case <-w.ctx.Done():
			return
		}
	}
}

// handleFileEvent 处理文件事件
func (w *Watcher) handleFileEvent(event fsnotify.Event) error {
	fileName := filepath.Base(event.Name)

	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		// 文件修改 - 重新加载策略
		log.Printf("Policy file modified: %s", fileName)
		return w.reloadPolicy(fileName, event.Name)

	case event.Op&fsnotify.Create == fsnotify.Create:
		// 文件创建 - 加载新策略
		log.Printf("Policy file created: %s", fileName)
		return w.loadPolicy(fileName, event.Name)

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// 文件删除 - 移除策略
		log.Printf("Policy file removed: %s", fileName)
		return w.removePolicy(fileName)

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// 文件重命名 - 移除旧策略
		log.Printf("Policy file renamed: %s", fileName)
		return w.removePolicy(fileName)
	}

	return nil
}

// loadPolicy 加载新策略
func (w *Watcher) loadPolicy(name string, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	// 验证策略
	if err := ValidatePolicy(string(content)); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// 加载策略
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := w.manager.LoadPolicy(ctx, name, string(content)); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	log.Printf("Successfully loaded policy: %s", name)
	return nil
}

// reloadPolicy 重新加载策略
func (w *Watcher) reloadPolicy(name string, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	// 验证策略
	if err := ValidatePolicy(string(content)); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// 重新加载策略
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 检查策略是否已存在
	if _, err := w.manager.GetPolicy(name); err == nil {
		// 策略存在，使用更新方法
		if err := w.manager.UpdatePolicy(ctx, name, &Policy{
			Content: string(content),
			Path:    path,
		}); err != nil {
			return fmt.Errorf("failed to update policy: %w", err)
		}
	} else {
		// 策略不存在，加载新策略
		if err := w.manager.LoadPolicy(ctx, name, string(content)); err != nil {
			return fmt.Errorf("failed to load policy: %w", err)
		}
	}

	log.Printf("Successfully reloaded policy: %s", name)
	return nil
}

// removePolicy 移除策略
func (w *Watcher) removePolicy(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := w.manager.RemovePolicy(ctx, name); err != nil {
		return fmt.Errorf("failed to remove policy: %w", err)
	}

	log.Printf("Successfully removed policy: %s", name)
	return nil
}

// IsRunning 检查监听器是否正在运行
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// GetWatchDir 获取监听目录
func (w *Watcher) GetWatchDir() string {
	return w.watchDir
}
