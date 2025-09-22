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

package nats

import (
	"strconv"
	"strings"

	"github.com/innovationmech/swit/pkg/messaging"
)

// validateJetStream performs semantic validation for JetStream configuration.
// It returns adapter-style errors, warnings, and suggestions.
func validateJetStream(js *JetStreamConfig) (
	[]messaging.AdapterValidationError,
	[]messaging.AdapterValidationWarning,
	[]messaging.AdapterValidationSuggestion,
) {
	var errs []messaging.AdapterValidationError
	var warns []messaging.AdapterValidationWarning
	var suggs []messaging.AdapterValidationSuggestion

	if js == nil || !js.Enabled {
		return errs, warns, suggs
	}

	// If enabled but no topology configured, add a warning (not fatal)
	if len(js.Streams) == 0 && len(js.Consumers) == 0 && len(js.KV) == 0 && len(js.ObjectStores) == 0 {
		warns = append(warns, messaging.AdapterValidationWarning{
			Field:   "Extra.nats.jetstream",
			Message: "JetStream enabled but no streams/consumers/KV/ObjectStores declared",
			Code:    "NATS_JS_EMPTY_TOPOLOGY",
		})
	}

	validRetention := map[string]struct{}{"limits": {}, "workqueue": {}, "interest": {}}
	validStorage := map[string]struct{}{"file": {}, "memory": {}}
	validDeliverPolicy := map[string]struct{}{"all": {}, "last": {}, "new": {}, "by_start_sequence": {}, "by_start_time": {}}
	validAckPolicy := map[string]struct{}{"none": {}, "all": {}, "explicit": {}}
	validReplayPolicy := map[string]struct{}{"instant": {}, "original": {}}

	// Streams validation
	for i, s := range js.Streams {
		if strings.TrimSpace(s.Name) == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "].name",
				Message:  "stream name is required",
				Code:     "NATS_JS_STREAM_NAME_REQUIRED",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if len(s.Subjects) == 0 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "].subjects",
				Message:  "at least one subject is required",
				Code:     "NATS_JS_STREAM_SUBJECTS_REQUIRED",
				Severity: messaging.AdapterValidationSeverityError,
			})
		} else {
			for si, sub := range s.Subjects {
				if strings.TrimSpace(sub) == "" {
					errs = append(errs, messaging.AdapterValidationError{
						Field:    "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "].subjects[" + strconv.Itoa(si) + "]",
						Message:  "subject cannot be empty",
						Code:     "NATS_JS_STREAM_SUBJECT_EMPTY",
						Severity: messaging.AdapterValidationSeverityError,
					})
				}
			}
		}
		if _, ok := validRetention[strings.ToLower(s.Retention)]; s.Retention != "" && !ok {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "].retention",
				Message:  "invalid retention; must be one of limits|workqueue|interest",
				Code:     "NATS_JS_STREAM_RETENTION_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if s.MaxBytes < 0 || s.MaxMsgs < 0 || s.MaxMsgSize < 0 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "]",
				Message:  "limits (max_bytes/max_msgs/max_msg_size) cannot be negative",
				Code:     "NATS_JS_STREAM_LIMITS_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if s.Replicas < 0 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "].replicas",
				Message:  "replicas cannot be negative",
				Code:     "NATS_JS_STREAM_REPLICAS_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		} else if s.Replicas == 0 {
			suggs = append(suggs, messaging.AdapterValidationSuggestion{
				Field:          "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "].replicas",
				Message:        "replicas defaults to 1",
				SuggestedValue: 1,
			})
		}
		if s.Storage != "" {
			if _, ok := validStorage[strings.ToLower(s.Storage)]; !ok {
				errs = append(errs, messaging.AdapterValidationError{
					Field:    "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "].storage",
					Message:  "invalid storage; must be one of file|memory",
					Code:     "NATS_JS_STREAM_STORAGE_INVALID",
					Severity: messaging.AdapterValidationSeverityError,
				})
			}
		} else {
			suggs = append(suggs, messaging.AdapterValidationSuggestion{
				Field:          "Extra.nats.jetstream.streams[" + strconv.Itoa(i) + "].storage",
				Message:        "default storage is file",
				SuggestedValue: "file",
			})
		}
	}

	// Consumers validation
	for i, c := range js.Consumers {
		if strings.TrimSpace(c.Name) == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.consumers[" + strconv.Itoa(i) + "].name",
				Message:  "consumer name is required",
				Code:     "NATS_JS_CONSUMER_NAME_REQUIRED",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if strings.TrimSpace(c.Stream) == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.consumers[" + strconv.Itoa(i) + "].stream",
				Message:  "stream is required for consumer",
				Code:     "NATS_JS_CONSUMER_STREAM_REQUIRED",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if c.DeliverPolicy != "" {
			if _, ok := validDeliverPolicy[strings.ToLower(c.DeliverPolicy)]; !ok {
				errs = append(errs, messaging.AdapterValidationError{
					Field:    "Extra.nats.jetstream.consumers[" + strconv.Itoa(i) + "].deliver_policy",
					Message:  "invalid deliver_policy",
					Code:     "NATS_JS_CONSUMER_DELIVER_POLICY_INVALID",
					Severity: messaging.AdapterValidationSeverityError,
				})
			}
		}
		if c.AckPolicy != "" {
			if _, ok := validAckPolicy[strings.ToLower(c.AckPolicy)]; !ok {
				errs = append(errs, messaging.AdapterValidationError{
					Field:    "Extra.nats.jetstream.consumers[" + strconv.Itoa(i) + "].ack_policy",
					Message:  "invalid ack_policy",
					Code:     "NATS_JS_CONSUMER_ACK_POLICY_INVALID",
					Severity: messaging.AdapterValidationSeverityError,
				})
			}
		}
		if c.ReplayPolicy != "" {
			if _, ok := validReplayPolicy[strings.ToLower(c.ReplayPolicy)]; !ok {
				errs = append(errs, messaging.AdapterValidationError{
					Field:    "Extra.nats.jetstream.consumers[" + strconv.Itoa(i) + "].replay_policy",
					Message:  "invalid replay_policy",
					Code:     "NATS_JS_CONSUMER_REPLAY_POLICY_INVALID",
					Severity: messaging.AdapterValidationSeverityError,
				})
			}
		}
		if c.MaxAckPending < 0 || c.MaxDeliver < 0 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.consumers[" + strconv.Itoa(i) + "]",
				Message:  "max_ack_pending and max_deliver cannot be negative",
				Code:     "NATS_JS_CONSUMER_LIMITS_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
	}

	// KV validation
	for i, kv := range js.KV {
		if strings.TrimSpace(kv.Name) == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.kv[" + strconv.Itoa(i) + "].name",
				Message:  "kv bucket name is required",
				Code:     "NATS_JS_KV_NAME_REQUIRED",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if kv.History < 0 || kv.History > 64 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.kv[" + strconv.Itoa(i) + "].history",
				Message:  "history must be between 0 and 64",
				Code:     "NATS_JS_KV_HISTORY_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if kv.Replicas < 0 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.kv[" + strconv.Itoa(i) + "].replicas",
				Message:  "replicas cannot be negative",
				Code:     "NATS_JS_KV_REPLICAS_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		} else if kv.Replicas == 0 {
			suggs = append(suggs, messaging.AdapterValidationSuggestion{
				Field:          "Extra.nats.jetstream.kv[" + strconv.Itoa(i) + "].replicas",
				Message:        "replicas defaults to 1",
				SuggestedValue: 1,
			})
		}
		if kv.TTL < 0 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.kv[" + strconv.Itoa(i) + "].ttl",
				Message:  "ttl cannot be negative",
				Code:     "NATS_JS_KV_TTL_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if kv.Storage != "" {
			if _, ok := validStorage[strings.ToLower(kv.Storage)]; !ok {
				errs = append(errs, messaging.AdapterValidationError{
					Field:    "Extra.nats.jetstream.kv[" + strconv.Itoa(i) + "].storage",
					Message:  "invalid storage; must be one of file|memory",
					Code:     "NATS_JS_KV_STORAGE_INVALID",
					Severity: messaging.AdapterValidationSeverityError,
				})
			}
		}
	}

	// Object Store validation
	for i, osb := range js.ObjectStores {
		if strings.TrimSpace(osb.Name) == "" {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.object_stores[" + strconv.Itoa(i) + "].name",
				Message:  "object store bucket name is required",
				Code:     "NATS_JS_OBJ_NAME_REQUIRED",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if osb.Replicas < 0 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.object_stores[" + strconv.Itoa(i) + "].replicas",
				Message:  "replicas cannot be negative",
				Code:     "NATS_JS_OBJ_REPLICAS_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		} else if osb.Replicas == 0 {
			suggs = append(suggs, messaging.AdapterValidationSuggestion{
				Field:          "Extra.nats.jetstream.object_stores[" + strconv.Itoa(i) + "].replicas",
				Message:        "replicas defaults to 1",
				SuggestedValue: 1,
			})
		}
		if osb.MaxBucketBytes < 0 {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Extra.nats.jetstream.object_stores[" + strconv.Itoa(i) + "].max_bucket_bytes",
				Message:  "max_bucket_bytes cannot be negative",
				Code:     "NATS_JS_OBJ_LIMITS_INVALID",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
		if osb.Storage != "" {
			if _, ok := validStorage[strings.ToLower(osb.Storage)]; !ok {
				errs = append(errs, messaging.AdapterValidationError{
					Field:    "Extra.nats.jetstream.object_stores[" + strconv.Itoa(i) + "].storage",
					Message:  "invalid storage; must be one of file|memory",
					Code:     "NATS_JS_OBJ_STORAGE_INVALID",
					Severity: messaging.AdapterValidationSeverityError,
				})
			}
		}
	}

	return errs, warns, suggs
}

// validateNATSConfiguration performs endpoint and TLS-related checks:
// - Endpoints should use nats:// or tls-enabled scheme (nats, natsws etc. kept simple: expect scheme present)
// - Warn when TLS is enabled but SkipVerify=true
func validateNATSConfiguration(base *messaging.BrokerConfig, cfg *Config) (
	[]messaging.AdapterValidationError,
	[]messaging.AdapterValidationWarning,
	[]messaging.AdapterValidationSuggestion,
) {
	var errs []messaging.AdapterValidationError
	var warns []messaging.AdapterValidationWarning
	var suggs []messaging.AdapterValidationSuggestion

	for i, ep := range base.Endpoints {
		s := strings.TrimSpace(ep)
		// Require a scheme for clarity (e.g., nats://host:4222)
		if !strings.Contains(s, "://") {
			errs = append(errs, messaging.AdapterValidationError{
				Field:    "Endpoints[" + strconv.Itoa(i) + "]",
				Message:  "endpoint should include scheme, e.g., nats://host:4222",
				Code:     "NATS_ENDPOINT_SCHEME_REQUIRED",
				Severity: messaging.AdapterValidationSeverityError,
			})
		}
	}

	if cfg != nil && base != nil && base.TLS != nil && base.TLS.Enabled && base.TLS.SkipVerify {
		warns = append(warns, messaging.AdapterValidationWarning{
			Field:   "TLS.SkipVerify",
			Message: "TLS enabled with SkipVerify=true may be insecure; consider valid CA and server name",
			Code:    "NATS_TLS_SKIP_VERIFY",
		})
	}

	// OAuth2 specific checks for NATS when selected
	if base != nil && base.Authentication != nil && base.Authentication.Type == messaging.AuthTypeOAuth2 {
		// Allow either static token or client credentials flow
		if strings.TrimSpace(base.Authentication.Token) == "" {
			if strings.TrimSpace(base.Authentication.ClientID) == "" || strings.TrimSpace(base.Authentication.ClientSecret) == "" || strings.TrimSpace(base.Authentication.TokenURL) == "" {
				errs = append(errs, messaging.AdapterValidationError{
					Field:    "Authentication",
					Message:  "oauth2 requires either token or client_id/client_secret/token_url",
					Code:     "NATS_OAUTH2_CONFIG_INCOMPLETE",
					Severity: messaging.AdapterValidationSeverityError,
				})
			}
		}
	}

	return errs, warns, suggs
}
