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

package kafka

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/riferrei/srclient"

	"github.com/innovationmech/swit/pkg/messaging"
)

type schemaRegistryManager struct {
	mu      sync.RWMutex
	clients map[string]*schemaRegistryClient
}

func newSchemaRegistryManager() *schemaRegistryManager {
	return &schemaRegistryManager{clients: make(map[string]*schemaRegistryClient)}
}

func (m *schemaRegistryManager) getClient(cfg *messaging.SchemaRegistryConfig) (*schemaRegistryClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("schema registry configuration is required")
	}

	key := registryCacheKey(cfg)

	m.mu.RLock()
	if client, ok := m.clients[key]; ok {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	client, err := newSchemaRegistryClient(cfg)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if existing, ok := m.clients[key]; ok {
		m.mu.Unlock()
		return existing, nil
	}
	m.clients[key] = client
	m.mu.Unlock()

	return client, nil
}

func registryCacheKey(cfg *messaging.SchemaRegistryConfig) string {
	if cfg.Authentication == nil {
		return cfg.URL
	}

	var secret string
	switch cfg.Authentication.Type {
	case "basic":
		secret = cfg.Authentication.Password
	case "bearer":
		secret = cfg.Authentication.Token
	case "api-key":
		secret = cfg.Authentication.APIKey
	}

	digest := sha256.Sum256([]byte(secret))
	return strings.Join([]string{cfg.URL, cfg.Authentication.Type, cfg.Authentication.Username, hex.EncodeToString(digest[:])}, "|")
}

type schemaRegistryClient struct {
	client    srclient.ISchemaRegistryClient
	cacheSize int

	mu           sync.RWMutex
	subjectCache map[string]*schemaDescriptor
	idCache      map[int]*schemaDescriptor
}

type schemaDescriptor struct {
	subject string
	version int
	id      int
	format  messaging.SerializationFormat
	schema  *srclient.Schema
}

func newSchemaRegistryClient(cfg *messaging.SchemaRegistryConfig) (*schemaRegistryClient, error) {
	client := srclient.NewSchemaRegistryClient(cfg.URL)
	client.CodecCreationEnabled(true)
	client.CodecJsonEnabled(true)

	if cfg.Authentication != nil {
		switch cfg.Authentication.Type {
		case "basic":
			client.SetCredentials(cfg.Authentication.Username, cfg.Authentication.Password)
		case "bearer":
			client.SetBearerToken(cfg.Authentication.Token)
		case "api-key":
			client.SetCredentials(cfg.Authentication.APIKey, cfg.Authentication.APIKey)
		}
	}

	return &schemaRegistryClient{
		client:       client,
		cacheSize:    cfg.CacheSize,
		subjectCache: make(map[string]*schemaDescriptor),
		idCache:      make(map[int]*schemaDescriptor),
	}, nil
}

func (c *schemaRegistryClient) getSchemaForSubject(_ context.Context, subject, version string) (*schemaDescriptor, error) {
	if subject == "" {
		return nil, fmt.Errorf("schema registry subject is required")
	}

	cacheKey := cacheKey(subject, normalizeVersion(version))

	c.mu.RLock()
	if descriptor, ok := c.subjectCache[cacheKey]; ok {
		c.mu.RUnlock()
		return descriptor, nil
	}
	c.mu.RUnlock()

	schema, err := c.loadSchema(subject, version)
	if err != nil {
		return nil, err
	}

	descriptor, err := c.buildDescriptor(subject, schema)
	if err != nil {
		return nil, err
	}

	c.storeDescriptor(cacheKey, descriptor)
	return descriptor, nil
}

func (c *schemaRegistryClient) getSchemaByID(_ context.Context, id int) (*schemaDescriptor, error) {
	if id <= 0 {
		return nil, fmt.Errorf("schema id must be positive")
	}

	c.mu.RLock()
	if descriptor, ok := c.idCache[id]; ok {
		c.mu.RUnlock()
		return descriptor, nil
	}
	c.mu.RUnlock()

	schema, err := c.client.GetSchema(id)
	if err != nil {
		return nil, fmt.Errorf("schema registry lookup by id %d failed: %w", id, err)
	}

	descriptor, err := c.buildDescriptor("", schema)
	if err != nil {
		return nil, err
	}

	cacheKey := cacheKey(descriptor.subject, normalizeVersion(strconv.Itoa(descriptor.version)))
	c.storeDescriptor(cacheKey, descriptor)
	return descriptor, nil
}

func (c *schemaRegistryClient) loadSchema(subject, version string) (*srclient.Schema, error) {
	var (
		schema *srclient.Schema
		err    error
	)

	normVersion := normalizeVersion(version)
	if normVersion == "latest" {
		schema, err = c.client.GetLatestSchema(subject)
	} else {
		ver, convErr := strconv.Atoi(normVersion)
		if convErr != nil {
			return nil, fmt.Errorf("invalid schema version %q: %w", version, convErr)
		}
		schema, err = c.client.GetSchemaByVersion(subject, ver)
	}

	if err != nil {
		return nil, fmt.Errorf("schema registry lookup for subject %q version %q failed: %w", subject, version, err)
	}

	return schema, nil
}

func (c *schemaRegistryClient) buildDescriptor(subject string, schema *srclient.Schema) (*schemaDescriptor, error) {
	format, err := mapSchemaType(schema)
	if err != nil {
		return nil, err
	}

	d := &schemaDescriptor{
		subject: subject,
		version: schema.Version(),
		id:      schema.ID(),
		format:  format,
		schema:  schema,
	}

	return d, nil
}

func (c *schemaRegistryClient) storeDescriptor(cacheKey string, descriptor *schemaDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cacheSize > 0 && len(c.subjectCache) >= c.cacheSize {
		for key, cached := range c.subjectCache {
			delete(c.subjectCache, key)
			delete(c.idCache, cached.id)
			if len(c.subjectCache) < c.cacheSize {
				break
			}
		}
	}

	if descriptor.subject != "" && cacheKey != "" {
		c.subjectCache[cacheKey] = descriptor
	}
	c.idCache[descriptor.id] = descriptor
}

func mapSchemaType(schema *srclient.Schema) (messaging.SerializationFormat, error) {
	schemaType := schema.SchemaType()
	if schemaType == nil {
		return messaging.FormatAvro, nil
	}

	switch *schemaType {
	case srclient.Avro:
		return messaging.FormatAvro, nil
	case srclient.Json:
		return messaging.FormatJSON, nil
	case srclient.Protobuf:
		return messaging.FormatProtobuf, nil
	default:
		return "", fmt.Errorf("unsupported schema registry type: %s", string(*schemaType))
	}
}

func normalizeVersion(version string) string {
	if version == "" {
		return "latest"
	}
	if strings.EqualFold(version, "latest") {
		return "latest"
	}
	return version
}

func cacheKey(subject, version string) string {
	return subject + ":" + version
}
