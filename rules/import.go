package rules

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"rba/services"
	"rba/services/database"
	"rba/types"
	"rba/util"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Rules        []types.RuleConfig `yaml:"rules"`
	Services     ServicesConfig     `yaml:"services"`
	Auth         AuthConfig         `yaml:"auth"`
	EventStorage EventStorageConfig `yaml:"eventStorage"`
}

type EventStorageConfig struct {
	Enabled  bool    `yaml:"enabled"`
	MinScore float64 `yaml:"minScore" validate:"required_if=Enabled true,gte=0,lte=1"`
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	Method  string `yaml:"method"`
}

type ServicesConfig struct {
	Redis    RedisConfig    `yaml:"redis"`
	Nats     NatsConfig     `yaml:"nats"`
	GeoIP    GeoIPConfig    `yaml:"geoIP"`
	Database DatabaseConfig `yaml:"database"`
}

type DatabaseConfig struct {
	Enabled    bool      `yaml:"enabled"`
	Type       string    `yaml:"type"`
	Connection yaml.Node `yaml:"connection"`
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

type ctxRequestEventType struct{}

func WithRequestEventType(ctx context.Context, tenant string) context.Context {
	return context.WithValue(ctx, ctxRequestEventType{}, tenant)
}

func RequestEventTypeFromContext(ctx context.Context) (string, bool) {
	tenant, ok := ctx.Value(ctxRequestEventType{}).(string)
	return tenant, ok
}

func LoadConfig(path string) (map[string][]util.NamedRiskHandler, Config, error) {
	var handlers = make(map[string][]util.NamedRiskHandler)
	data, err := os.ReadFile(path)

	var cfg = Config{}

	if err != nil {
		return nil, cfg, err
	}

	// Parse the yaml into cfg. Then iterate through rules pushing to the provided parser
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Println(err)
		return nil, cfg, err
	}

	if cfg.Auth.Enabled {
		if cfg.Auth.Method != "hmac" && cfg.Auth.Method != "jwt" {
			panic("auth must be either hmac or jwt")
		}
	}

	// Parse Services and ensure connections setup.
	if cfg.Services.Nats.Enabled {
		if cfg.Services.Nats.Threshold < 0 || cfg.Services.Nats.Threshold > 1 {
			panic("Threshold for publishing must be between 0 and 1")
		}
		if cfg.Services.Nats.Url == "" {
			panic("Provide a valid nats URL")
		}
		_, err := services.ConnectNats(cfg.Services.Nats.Url)
		if err != nil {
			panic(err)
		}
	}

	if cfg.Services.Redis.Enabled {
		redisHost, ok := util.ConfigFallback(cfg.Services.Redis.Host, "REDIS_HOST")
		if !ok {
			panic("Provide a valid redis host")
		}
		_, err := services.ConnectRedis(redisHost)
		if err != nil {
			log.Println(err)
			panic("Could not connect to redis. Please check configuration")
		}
	}

	if cfg.Services.Database.Enabled {
		databaseType, ok := util.ConfigFallback(cfg.Services.Database.Type, "DATABASE_TYPE")
		if !ok {
			log.Fatal("Must provide a database type when enabled")
		}

		if err := database.ConnectDatabase(databaseType, cfg.Services.Database.Connection); err != nil {
			log.Fatal(err)
		}
	}

	eventStorageConfig := cfg.EventStorage
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(eventStorageConfig); err != nil {
		log.Fatal(err)
	}
	if eventStorageConfig.Enabled && !cfg.Services.Database.Enabled {
		log.Fatal("event storage requires a configured database")
	}

	if cfg.Services.GeoIP.Enabled {
		if cfg.Services.GeoIP.FileType != "mmdb" {
			panic("Currently only supporting mmdb filetypes for GeoIP configuration")
		}

		if cfg.Services.GeoIP.Download == true {
			if cfg.Services.GeoIP.Source.SourceType != "s3" {
				panic("GeoIP: Only s3 accepted for file source currently")
			}

			bucketKey, ok := util.ConfigFallback(cfg.Services.GeoIP.Source.BucketKey, "GEOIP_S3_BUCKET_KEY")
			if !ok {
				panic("GeoIP: Must provide a valid bucket key for s3 download")
			}

			bucketName, ok := util.ConfigFallback(cfg.Services.GeoIP.Source.BucketName, "GEOIP_S3_BUCKET_NAME")
			if !ok {
				panic("GeoIP: Must provide a valid bucket name for s3 download")
			}

			ctx := context.Background()
			if err := downloadS3File(ctx, bucketName, bucketKey, cfg.Services.GeoIP.Path); err != nil {
				log.Fatalf("Failed to download geoIP file: %s", err)
			}
		}
		// Init the db to ensure file validity
		_, err = services.InitCityDB(cfg.Services.GeoIP.Path)
		if err != nil {
			log.Fatalf("Could not open mmdb: %s", err)
		}
	}

	for _, rawRule := range cfg.Rules {
		switch rawRule.Name {
		case util.Rules.Velocity:
			handler, err := parseVelocityRule(rawRule.Params)
			if err != nil {
				return nil, cfg, err
			}
			handlers["login"] = append(handlers["login"], handler)
		case util.Rules.Denylist:
			handler, err := parseDenylistRule(rawRule.Config)
			if err != nil {
				return nil, cfg, err
			}
			handlers["login"] = append(handlers["login"], handler)
		case util.Rules.HorizontalBruteForce:
			handler, err := parseHorizontalBruteForceRule(rawRule.Params)
			if err != nil {
				return nil, cfg, err
			}
			handlers["login_failure"] = append(handlers["login_failure"], handler)
		case util.Rules.ImpossibleTravel:
			handler, err := parseImpossibleTravelRule(rawRule.Params)
			if err != nil {
				return nil, cfg, err
			}
			handlers["login"] = append(handlers["login"], handler)
		case util.Rules.RateLimitFailedLogins:
			handler, err := parseRateLimitFailedLoginsRule(rawRule.Params)
			if err != nil {
				return nil, cfg, err
			}
			handlers["login"] = append(handlers["login"], handler)
			handlers["login_failure"] = append(handlers["login_failure"], handler)
		}
	}

	return handlers, cfg, nil
}
