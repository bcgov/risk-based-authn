package rules

import (
	"context"
	"testing"

	"gopkg.in/yaml.v3"
)

var yamlTests = []struct {
	valid bool
	yaml  string
}{
	{
		yaml: `
sourceList: database
ips:
  - 1.2.3.4
cidrs:
  - 192.168.1.0/24
strategy: override`,
		valid: true,
	},
	{
		yaml: `
sourceList: database
strategy: override`,
		valid: true,
	},
	{
		yaml: `
sourceList: static
ips:
  - 1.2.3.4
strategy: override`,
		valid: true,
	},
	{
		// invalid sourceList
		yaml: `
sourceList: dne
strategy: average`,
		valid: false,
	},
	{
		// invalid strategy
		yaml: `
sourceList: database
strategy: dne`,
		valid: false,
	},
	{
		// invalid IP
		yaml: `
sourceList: database
ips: 
  - string
strategy: override`,
		valid: false,
	},
	{
		// invalid IP
		yaml: `
sourceList: database
ips: 
  - string
strategy: override`,
		valid: false,
	},
	{
		// invalid CIDR
		yaml: `
sourceList: database
cidrs: 
  - string
strategy: override`,
		valid: false,
	},
	{
		// invalid CIDR
		yaml: `
sourceList: database
cidrs: 
  - 1.2.3.4
strategy: override`,
		valid: false,
	},
}

func TestDenyListConfigYAML(t *testing.T) {
	for _, testCase := range yamlTests {
		var node yaml.Node
		yaml.Unmarshal([]byte(testCase.yaml), &node)
		_, err := parseDenylistRule(node)
		if testCase.valid && err != nil {
			t.Fatalf("validation failed: %v", err)
		}
		if !testCase.valid && err == nil {
			t.Fatalf("validation passed on invalid yaml: %s", testCase.yaml)
		}
	}
}

func TestDenyListRule(t *testing.T) {
	config := `
sourceList: static
cidrs:
  - 1.2.3.0/24
strategy: override`

	var node yaml.Node
	yaml.Unmarshal([]byte(config), &node)
	handler, _ := parseDenylistRule(node)

	raw := map[string]interface{}{
		"wrong": "payload",
	}

	result := handler.Handler(context.Background(), raw)
	if !(*result.Err == "missing ip") {
		t.Fatalf("unexpected error message")
	}

	raw = map[string]interface{}{
		"ip": "1.2.3.4",
	}

	result = handler.Handler(context.Background(), raw)
	if !(result.Score == 1) {
		t.Fatalf("unexpected score for ip")
	}

	raw = map[string]interface{}{
		"ip": "1.2.3.5",
	}

	result = handler.Handler(context.Background(), raw)
	if !(result.Score == 1) {
		t.Fatalf("unexpected score")
	}

	// Falls outside of provided CIDR
	raw = map[string]interface{}{
		"ip": "1.2.4.5",
	}

	result = handler.Handler(context.Background(), raw)
	if !(result.Score == 0) {
		t.Fatalf("unexpected score")
	}
}
