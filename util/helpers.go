package util

import (
	"encoding/json"
	"fmt"
	"log"
	"rba/services"
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
