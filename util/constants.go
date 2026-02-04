package util

import "errors"

type serviceConstants struct {
	Redis    string
	Nats     string
	Database string
}

var Services = serviceConstants{
	Redis:    "redis",
	Nats:     "nats",
	Database: "database",
}

type rules struct {
	Denylist              string
	Velocity              string
	HorizontalBruteForce  string
	ImpossibleTravel      string
	RateLimitFailedLogins string
}

var Rules = rules{
	Denylist:              "denylist",
	Velocity:              "velocity",
	HorizontalBruteForce:  "horizontalBruteForce",
	ImpossibleTravel:      "impossibleTravel",
	RateLimitFailedLogins: "rateLimitFailedLogins",
}

type strategies struct {
	Override string
	Average  string
}

var Strategies = strategies{
	Override: "override",
	Average:  "average",
}

type database struct {
	Postgres string
	Mongo    string
}

var Databases = database{
	Postgres: "postgres",
	Mongo:    "mongo",
}

var (
	ErrInvalidNetwork       = errors.New("invalid network")
	ErrNetworkAlreadyExists = errors.New("network already exists")
	ErrUnknown              = errors.New("unknown server error")
)
