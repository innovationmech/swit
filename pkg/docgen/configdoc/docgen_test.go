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
