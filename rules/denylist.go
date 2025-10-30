package rules

import (
	"context"
	"errors"
	"fmt"
	"net"
	"rba/util"
	"slices"
)

// Checks if IP in CIDR range, or directly equal
func ipInCIDR(ipStr, cidrOrIPStr string) (bool, error) {
	userIP := net.ParseIP(ipStr)
	if userIP == nil {
		return false, fmt.Errorf("invalid IP: %s", ipStr)
	}

	_, ipNet, err := net.ParseCIDR(cidrOrIPStr)
	if err == nil {
		return ipNet.Contains(userIP), nil
	}

	targetIP := net.ParseIP(cidrOrIPStr)
	if targetIP == nil {
		return false, fmt.Errorf("invalid CIDR or IP: %s", cidrOrIPStr)
	}
	return userIP.Equal(targetIP), nil
}

func parseDenylistRule(raw map[string]interface{}) (util.NamedRiskHandler, error) {
	ipsRaw, ok := raw["ips"]
	if !ok {
		return util.NamedRiskHandler{}, errors.New("denylist: missing or invalid ip list")
	}

	rawList, ok := ipsRaw.([]interface{})
	if !ok {
		return util.NamedRiskHandler{}, errors.New("denylist: denylistedIPs must be a list")
	}

	strategy, ok := raw["strategy"].(string)
	if !ok || !slices.Contains(util.Strategies, strategy) {
		return util.NamedRiskHandler{}, errors.New("denylist: missing or invalid strategy")
	}

	// Convert []interface{} to []string
	ips := make([]string, 0, len(rawList))

	for _, item := range rawList {
		ipOrCIDR, ok := item.(string)
		if !ok {
			return util.NamedRiskHandler{}, errors.New("denylist: denylistedIPs mustbe strings")
		}
		_, ipNet, cidrParseErr := net.ParseCIDR(ipOrCIDR)
		if cidrParseErr == nil {
			ips = append(ips, ipNet.String())
			continue
		}
		targetIP := net.ParseIP(ipOrCIDR)
		if targetIP == nil {
			return util.NamedRiskHandler{}, errors.New("denylist: invalid ip address provided, provide an ip or cidr")
		}
		ips = append(ips, targetIP.String())
	}

	return util.NamedRiskHandler{
		Name:     "denylist",
		Strategy: strategy,
		Handler: func(ctx context.Context, args map[string]interface{}) util.RiskResult {
			base := util.RiskResult{
				Name:     "denylist",
				Strategy: strategy,
				Score:    0,
				Err:      nil,
			}

			ip, err := util.GetStringField(args, "ip")

			if err != nil {
				errText := "missing ip"
				result := base
				result.Err = &errText
				return result
			}

			for _, blockedIp := range ips {
				inRange, err := ipInCIDR(ip, blockedIp)
				if err != nil {
					errText := err.Error()
					result := base
					result.Err = &errText
					return result
				}
				if inRange {
					result := base
					result.Score = 1
					return result
				}

			}

			result := base
			return result
		},
	}, nil
}
