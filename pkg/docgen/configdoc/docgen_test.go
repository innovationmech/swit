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

package configdoc

import (
	"reflect"
	"strings"
	"testing"

	"github.com/innovationmech/swit/pkg/server"
)

func TestGenerate_ServerConfigMinimal(t *testing.T) {
	sc := server.NewServerConfig()
	sc.SetDefaults()
	md, err := Generate([]Target{{Title: "ServerConfig", Type: reflect.TypeOf(*sc), DefaultValue: reflect.ValueOf(sc).Elem()}})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !strings.Contains(md, "## Configuration Reference (generated)") {
		t.Fatalf("missing header in output: %s", md[:100])
	}
	if !strings.Contains(md, "http.port") {
		t.Fatalf("expected http.port key in output")
	}
	if !strings.Contains(md, "grpc.port") {
		t.Fatalf("expected grpc.port key in output")
	}
}
