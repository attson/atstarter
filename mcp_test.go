package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPInitializeUsesRootyardServerName(t *testing.T) {
	response := (&mcpServer{}).handle(mcpRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
	})
	result := response.Result.(map[string]interface{})
	info := result["serverInfo"].(map[string]interface{})
	if info["name"] != "rootyard" {
		t.Fatalf("server name = %v, want rootyard", info["name"])
	}
}

func TestRootyardMCPToolsUseOnlyRootyardPrefix(t *testing.T) {
	tools := rootyardMCPTools()
	if len(tools) == 0 {
		t.Fatal("tool list is empty")
	}
	for _, tool := range tools {
		if !strings.HasPrefix(tool.Name, "rootyard_") {
			t.Errorf("tool %q does not use rootyard_ prefix", tool.Name)
		}
		if strings.HasPrefix(tool.Name, "atstarter_") {
			t.Errorf("legacy tool registered: %q", tool.Name)
		}
	}
}
