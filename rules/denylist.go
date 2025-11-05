package rules

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"rba/services"
	"rba/types"
	"rba/util"

	"github.com/redis/go-redis/v9"
)

/*
	TODO: since all config changes are in this package, rule config should be kept here in mem instead of passed throughout the server.
*/

func UpdateDenylistParam(rules []types.RuleConfig, ip string, paramType string, operation string) error {
	if paramType != "ip" && paramType != "cidr" {
		return errors.New("must provide cidr or ip for the param type")
	}

	if operation != "add" && operation != "remove" {
		return errors.New("must provide add or remove for the operation")
	}

	denylistParams, parseErr := util.GetRuleConfig(rules, "denylist")
	if parseErr != nil {
		return errors.New("denylist not in configuration")
	}

	sourceList, ok := denylistParams.Params["sourceList"]
	if !ok {
		return errors.New("denylist misconfigured")
	}

	if sourceList == util.Services.Redis {
		ctx := context.TODO()
		var ipCmd *redis.IntCmd
		if operation == "add" {
			ipCmd = services.RedisClient.SAdd(ctx, "denylist:"+paramType+"s", ip)
		} else {
			ipCmd = services.RedisClient.SRem(ctx, "denylist:"+paramType+"s", ip)
		}
		_, err := ipCmd.Result()
		if err != nil {
			return errors.New("failed to add param to redis")
		}
		return nil
	} else {
		return errors.New("no dynamic source configured")
	}
}

func GetDenylistParams(rules []types.RuleConfig, paramType string) ([]string, error) {
	denylistParams, parseErr := util.GetRuleConfig(rules, "denylist")
	if parseErr != nil {
		return nil, errors.New("denylist not in configuration")
	}

	sourceList, ok := denylistParams.Params["sourceList"]
	if !ok {
		return nil, errors.New("denylist misconfigured")
	}

	if sourceList == util.Services.Redis {
		ctx := context.TODO()
		ipCmd := services.RedisClient.SMembers(ctx, "denylist:"+paramType)
		result, err := ipCmd.Result()
		if err != nil {
			return nil, errors.New("failed to add param to redis")
		}
		return result, nil
	} else {
		list, ok := denylistParams.Params[paramType]
		log.Print(list)
		log.Printf("list type: %T\n", list)

		if !ok {
			// It is valid for only of of IP or CIDR to be configured, in which case it is an empty list
			return []string{}, nil
		}
		// YAML is always parsed as []interface{} for lists. Need to type check and convert to []string
		if rawList, ok := list.([]interface{}); ok {
			result := make([]string, len(rawList))
			for i, v := range rawList {
				str, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("element %d in %s is not a string", i, paramType)
				}
				result[i] = str
			}
			return result, nil
		} else {
			return nil, errors.New("list is misconfigured")
		}
	}
}

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
	ipsRaw, ipsExist := raw["ips"]
	cidrsRaw, cidrsExist := raw["cidrs"]
	sourceList, sourceListExists := raw["sourceList"]

	if !sourceListExists {
		return util.NamedRiskHandler{}, errors.New("denylist: must provide source")
	}

	if sourceList == "static" && !ipsExist && !cidrsExist {
		return util.NamedRiskHandler{}, errors.New("denylist: must provide a static ip list when source is static")
	}

	ipsList, ok := ipsRaw.([]interface{})
	if ipsExist && !ok {
		return util.NamedRiskHandler{}, errors.New("denylist: denylisted IPs must be a list")
	}

	cidrList, ok := cidrsRaw.([]interface{})
	if cidrsExist && !ok {
		return util.NamedRiskHandler{}, errors.New("denylist: denylisted CIDRs must be a list")
	}

	strategy, ok := raw["strategy"].(string)
	if !ok || !util.IsValidStrategy(strategy) {
		return util.NamedRiskHandler{}, errors.New("denylist: missing or invalid strategy")
	}

	// Convert []interface{} to []string
	ips := make([]string, 0, len(ipsList))
	cidrs := make([]string, 0, len(cidrList))

	for _, item := range cidrList {
		cidr, ok := item.(string)
		if !ok {
			return util.NamedRiskHandler{}, errors.New("denylist: denylistedIPs must be strings")
		}
		_, ipNet, cidrParseErr := net.ParseCIDR(cidr)
		if cidrParseErr != nil {
			return util.NamedRiskHandler{}, errors.New("denylist: could not parse CIDR")
		}
		cidrs = append(cidrs, ipNet.String())
		if sourceList == util.Services.Redis {
			ctx := context.TODO()
			services.RedisClient.SAdd(ctx, "denylist:cidrs", cidrs)
		}
	}

	for _, item := range ipsList {
		ip, ok := item.(string)
		if !ok {
			return util.NamedRiskHandler{}, errors.New("denylist: denylistedIPs must be strings")
		}
		targetIP := net.ParseIP(ip)
		if targetIP == nil {
			return util.NamedRiskHandler{}, errors.New("denylist: invalid ip address provided, provide an ip")
		}
		ips = append(ips, targetIP.String())
		if sourceList == util.Services.Redis {
			ctx := context.TODO()
			services.RedisClient.SAdd(ctx, "denylist:ips", ips)
		}
	}

	return util.NamedRiskHandler{
		Name:     util.Rules.Denylist,
		Strategy: strategy,
		Handler: func(ctx context.Context, args map[string]interface{}) util.RiskResult {
			base := util.RiskResult{
				Name:     util.Rules.Denylist,
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
