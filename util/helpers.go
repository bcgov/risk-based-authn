package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"rba/services"
	"rba/types"
)

func GetStringField(m map[string]interface{}, key string) (string, error) {
	val, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing key: %s", key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("key %s is not a string", key)
	}
	return str, nil
}

func CalculateRisk(resultsChan <-chan RiskResult) (float64, []RiskResult) {
	var results []RiskResult
	var sum float64
	var count int
	var override bool

	for result := range resultsChan {
		results = append(results, result)
		if result.Err == nil {
			switch result.Strategy {
			case "average":
				sum += result.Score
				count++
			// Don't actually want to short-circuit here since we want the detailed breakdown in the response
			case "override":
				if result.Score == 1 {
					override = true
				}
			}
		}
	}

	var riskResult float64
	if override {
		riskResult = 1
	} else if count > 0 {
		riskResult = sum / float64(count)
	} else {
		riskResult = 0.0
	}
	return riskResult, results
}

func PublishMessage(results []RiskResult) {
	data, jsonErr := json.Marshal(results)
	if jsonErr != nil {
		log.Printf("Error getting NATS connection")
		return
	}

	err := services.NatsConn.Publish("alerts", data)
	if err != nil {
		log.Printf("NATS publish error: %v", err)
	}
}

func IsValidStrategy(val string) bool {
	switch val {
	case Strategies.Override, Strategies.Average:
		return true
	default:
		return false
	}
}

func GetRuleConfig(rules []types.RuleConfig, name string) (types.RuleConfig, error) {
	for _, rule := range rules {
		if rule.Name == name {
			return rule, nil
		}
	}
	return types.RuleConfig{}, errors.New("not found")
}

func ConfigFallback(yamlValue, envKey string) (string, bool) {
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue, true
	}
	if yamlValue != "" {
		return yamlValue, true
	}
	return "", false
}

// Check the provided value or an Envirornment variable fallback is provided. Exits with provided message if the value does not exist
func CheckConfigValueExists(value string, envName string, message string) string {
	checkedValue, ok := ConfigFallback(value, envName)
	if !ok {
		panic(message)
	}
	return checkedValue
}

/*
Convert app errors into correct status code for response. See constants.go for error definitions.
*/
func HttpStatusCodeForError(err error) int {
	switch {
	case errors.Is(err, ErrInvalidNetwork):
		return http.StatusBadRequest
	case errors.Is(err, ErrNetworkAlreadyExists):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
