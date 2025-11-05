package types

type RuleConfig struct {
	Name   string                 `yaml:"name"`
	Params map[string]interface{} `yaml:",inline"`
}
