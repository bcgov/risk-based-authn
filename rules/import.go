package rules

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"rba/services"
	"rba/types"
	"rba/util"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Rules    []types.RuleConfig `yaml:"rules"`
	Services ServicesConfig     `yaml:"services"`
	Auth     AuthConfig         `yaml:"auth"`
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	Method  string `yaml:"method"`
}

type ServicesConfig struct {
	Redis RedisConfig `yaml:"redis"`
	Nats  NatsConfig  `yaml:"nats"`
	GeoIP GeoIPConfig `yaml:"geoIP"`
}

type GeoIPConfig struct {
	FileType string     `yaml:"fileType"`
	Path     string     `yaml:"path"`
	Enabled  bool       `yaml:"enabled"`
	Download bool       `yaml:"download"`
	Source   FileSource `yaml:"source"`
}

type FileSource struct {
	SourceType string `yaml:"sourceType"`
	BucketName string `yaml:"bucketName"`
	BucketKey  string `yaml:"bucketKey"`
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

type Rule struct {
	Name string
}

func downloadS3File(ctx context.Context, bucketName string, objectKey string, fileName string) error {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("Couldn't load default configuration: %s", err)
		return err
	}

	s3Client := s3.NewFromConfig(sdkConfig)

	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		var noKey *s3types.NoSuchKey
		if errors.As(err, &noKey) {
			log.Printf("Can't get object %s from bucket %s. No such key exists.\n", objectKey, bucketName)
			err = noKey
		} else {
			log.Printf("Couldn't get object %v:%v. Here's why: %v\n", bucketName, objectKey, err)
		}
		return err
	}

	defer result.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("Couldn't create file %v. Here's why: %v\n", fileName, err)
		return err
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Couldn't read object body from %v. Here's why: %v\n", objectKey, err)
	}
	_, err = file.Write(body)
	return err
}

func LoadConfig(path string) (map[string][]util.NamedRiskHandler, ServicesConfig, AuthConfig, error) {
	var handlers = make(map[string][]util.NamedRiskHandler)
	data, err := os.ReadFile(path)

	var servicesConfig = ServicesConfig{}
	var authConfig = AuthConfig{}

	if err != nil {
		return nil, servicesConfig, authConfig, err
	}

	// Parse the yaml into cfg. Then iterate through rules pushing to the provided parser
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Println(err)
		return nil, servicesConfig, authConfig, err
	}

	if cfg.Auth.Enabled {
		if cfg.Auth.Method != "hmac" && cfg.Auth.Method != "jwt" {
			panic("auth must be either hmac or jwt")
		}
	}

	authConfig = cfg.Auth
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
		_, err := services.ConnectRedis(servicesConfig.Redis.Host)
		if err != nil {
			log.Println(err)
			panic("Could not connect to redis. Please check configuration")
		}
	}

	if servicesConfig.GeoIP.Enabled {
		if servicesConfig.GeoIP.FileType != "mmdb" {
			panic("Currently only supporting mmdb filetypes for GeoIP configuration")
		}

		if servicesConfig.GeoIP.Download == true {
			if servicesConfig.GeoIP.Source.SourceType != "s3" {
				panic("GeoIP: Only s3 accepted for file source currently")
			}

			if servicesConfig.GeoIP.Source.BucketKey == "" {
				panic("GeoIP: Must provide a valid bucket key for s3 download")
			}

			if servicesConfig.GeoIP.Source.BucketName == "" {
				panic("GeoIP: Must provide a valid bucket name for s3 download")
			}

			ctx := context.Background()
			if err := downloadS3File(ctx, servicesConfig.GeoIP.Source.BucketName, servicesConfig.GeoIP.Source.BucketKey, servicesConfig.GeoIP.Path); err != nil {
				log.Fatalf("Failed to download geoIP file: %s", err)
			}
		}
		// Init the db to ensure file validity
		_, err = services.InitCityDB(servicesConfig.GeoIP.Path)
		if err != nil {
			log.Fatalf("Could not open mmdb: %s", err)
		}
	}

	for _, rawRule := range cfg.Rules {
		switch rawRule.Name {
		case util.Rules.Velocity:
			handler, err := parseVelocityRule(rawRule.Params)
			if err != nil {
				return nil, servicesConfig, authConfig, err
			}
			handlers["login"] = append(handlers["login"], handler)
		case util.Rules.Denylist:
			handler, err := parseDenylistRule(rawRule.Params)
			if err != nil {
				return nil, servicesConfig, authConfig, err
			}
			handlers["login"] = append(handlers["login"], handler)
		case util.Rules.HorizontalBruteForce:
			handler, err := parseHorizontalBruteForceRule(rawRule.Params)
			if err != nil {
				return nil, servicesConfig, authConfig, err
			}
			handlers["login_failure"] = append(handlers["login_failure"], handler)
		case util.Rules.ImpossibleTravel:
			handler, err := parseImpossibleTravelRule(rawRule.Params)
			if err != nil {
				return nil, servicesConfig, authConfig, err
			}
			handlers["login"] = append(handlers["login"], handler)
		}
	}

	return handlers, servicesConfig, authConfig, nil
}
