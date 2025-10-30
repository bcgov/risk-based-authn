package util

import "context"

type RiskResult struct {
	Name     string
	Score    float64
	Strategy string
	Err      *string `json:"Err,omitempty"`
}

type NamedRiskHandler struct {
	Name     string
	Handler  RiskHandlerFunc
	Strategy string
}
type RiskHandlerFunc func(ctx context.Context, args map[string]interface{}) RiskResult
