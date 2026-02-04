package types

import "gopkg.in/yaml.v3"

type RuleConfig struct {
	Name   string                 `yaml:"name"`
	Params map[string]interface{} `yaml:",inline"`
	Config yaml.Node              `yaml:"config"`
}

type DatabaseConfig struct {
	Name       string    `yaml:"name"`
	Type       string    `yaml:"type"`
	Connection yaml.Node `yaml:"connection"`
}
