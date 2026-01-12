package services

import (
	"sync"

	"github.com/oschwald/geoip2-golang"
)

var (
	once      sync.Once
	CityDB    *geoip2.Reader
	mmdbError error
)

type FileSource struct {
	sourceType string
	bucketName string
	bucketKey  string
}

func InitCityDB(path string) (*geoip2.Reader, error) {
	once.Do(func() {
		CityDB, mmdbError = geoip2.Open(path)
	})
	return CityDB, mmdbError
}
