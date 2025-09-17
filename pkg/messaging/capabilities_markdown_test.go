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
	"strings"
	"testing"
)

func TestGenerateCapabilitiesMarkdown(t *testing.T) {
	md, err := GenerateCapabilitiesMarkdown()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if md == "" {
		t.Fatalf("expected non-empty markdown output")
	}
	// Basic sanity checks
	if !strings.Contains(md, "# Message Broker Capabilities Matrix") {
		t.Errorf("missing title header")
	}
	if !strings.Contains(md, "| Feature | Kafka | RabbitMQ | NATS |") {
		t.Errorf("missing feature table header")
	}
	if !strings.Contains(md, "Transactions") {
		t.Errorf("missing transactions row")
	}
	if !strings.Contains(md, "Graceful Degradation") {
		t.Errorf("missing graceful degradation section")
	}
}
