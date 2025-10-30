package rules

import (
	"log"
	"os"
	"rba/services"
	"rba/util"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Rules    []RuleConfig   `yaml:"rules"`
	Services ServicesConfig `yaml:"services"`
}

type ServicesConfig struct {
	Redis RedisConfig `yaml:"redis"`
	Nats  NatsConfig  `yaml:"nats"`
}

type NatsConfig struct {
	Url       string  `yaml:"url"`
	Threshold float64 `yaml:"threshold"`
	Enabled   bool    `yaml:"enabled"`
}

type RedisConfig struct {
	Host    string `yaml:"host"`
	Enabled bool   `yaml:"enabled"`
}

type RuleConfig struct {
	// Ah the missing piece. This yaml syntax is saying to push the name key into Name, which is how the case changes
	Name   string                 `yaml:"name"`
	Params map[string]interface{} `yaml:",inline"`
}

type Rule struct {
	Name string
}

func LoadConfig(path string) (map[string][]util.NamedRiskHandler, ServicesConfig, error) {
	var handlers = make(map[string][]util.NamedRiskHandler)
	data, err := os.ReadFile(path)

	var servicesConfig = ServicesConfig{}

	if err != nil {
		return nil, servicesConfig, err
	}

	// Parse the yaml into cfg. Then iterate through rules pushing to the provided parser
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Println(err)
		return nil, servicesConfig, err
	}

	servicesConfig = cfg.Services

	// Parse Services and ensure connections setup.
	if servicesConfig.Nats.Enabled {
		if servicesConfig.Nats.Threshold < 0 || servicesConfig.Nats.Threshold > 1 {
			panic("Threshold for publishing must be between 0 and 1")
		}
		if servicesConfig.Nats.Url == "" {
			panic("Provide a valid nats URL")
		}
		_, err := services.ConnectNats(servicesConfig.Nats.Url)
		if err != nil {
			panic(err)
		}
	}

	if servicesConfig.Redis.Enabled {
		if servicesConfig.Redis.Host == "" {
			panic("Provide a valid redis host")
		}
		redisConn := services.ConnectRedis(servicesConfig.Redis.Host)
		if redisConn == nil {
			panic("Could not connect to redis")
		}
	}

	for _, rawRule := range cfg.Rules {
		switch rawRule.Name {
		case "velocity":
			handler, err := parseVelocityRule(rawRule.Params)
			if err != nil {
				return nil, servicesConfig, err
			}
			handlers["login"] = append(handlers["login"], handler)
		case "denylist":
			handler, err := parseDenylistRule(rawRule.Params)
			if err != nil {
				return nil, servicesConfig, err
			}
			handlers["login"] = append(handlers["login"], handler)
		}

	}

	return handlers, servicesConfig, nil
}
