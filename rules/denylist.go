package rules

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"rba/services"
	"rba/util"

	"github.com/redis/go-redis/v9"
)

// Read-only configuration, should not be changed after initial parse. e.g. checking the sourceList to know to use redis.
type denylistConfigT struct {
	configured bool
	sourceList string
	ips        []string
	cidrs      []string
}

var denylistConfig = denylistConfigT{}

func UpdateDenylistParam(ctx context.Context, ip string, paramType string, operation string) (int, error) {
	if paramType != "ip" && paramType != "cidr" {
		return http.StatusBadRequest, errors.New("must provide cidr or ip for the param type")
	}

	if operation != "add" && operation != "remove" {
		return http.StatusBadRequest, errors.New("must provide add or remove for the operation")
	}

	if !denylistConfig.configured {
		return http.StatusBadRequest, errors.New("denylist is not configured")
	}

	if denylistConfig.sourceList == util.Services.Redis {
		if paramType == "ip" {
			targetIP := net.ParseIP(ip)
			if targetIP == nil {
				return http.StatusBadRequest, errors.New("invalid ip address")
			}
		}

		if paramType == "cidr" {
			_, _, err := net.ParseCIDR(ip)
			if err != nil {
				return http.StatusBadRequest, errors.New("invalid cidr")
			}
		}

		ctx := context.TODO()
		var ipCmd *redis.IntCmd
		if operation == "add" {
			ipCmd = services.RedisClient.SAdd(ctx, "denylist:"+paramType+"s", ip)
		} else {
			ipCmd = services.RedisClient.SRem(ctx, "denylist:"+paramType+"s", ip)
		}
		_, err := ipCmd.Result()
		if err != nil {
			return http.StatusInternalServerError, errors.New("failed to add param to redis")
		}
		return http.StatusOK, nil
	} else {
		return http.StatusBadRequest, errors.New("no dynamic source configured")
	}
}

func GetDenylistParams(ctx context.Context, paramType string) ([]string, int, error) {
	if paramType != "ips" && paramType != "cidrs" {
		return nil, http.StatusBadRequest, errors.New("must provide cidr or ip for the param type")
	}

	if !denylistConfig.configured {
		return nil, http.StatusBadRequest, errors.New("denylist is not configured")
	}

	if denylistConfig.sourceList == util.Services.Redis {
		ctx := context.TODO()
		ipCmd := services.RedisClient.SMembers(ctx, "denylist:"+paramType)
		result, err := ipCmd.Result()
		if err != nil {
			return nil, http.StatusInternalServerError, errors.New("failed to fetch list from redis")
		}
		return result, http.StatusOK, nil
	} else {
		if paramType == "ips" {
			return denylistConfig.ips, http.StatusOK, nil
		}
		return denylistConfig.cidrs, http.StatusOK, nil
	}
}

func RemoveDenylistEntry(ctx context.Context, paramType string, entry string) (int, error) {
	if paramType != "ip" && paramType != "cidr" {
		return http.StatusBadRequest, errors.New("must provide cidr or ip for the param type")
	}

	if !denylistConfig.configured {
		return http.StatusBadRequest, errors.New("denylist is not configured")
	}

	if denylistConfig.sourceList == util.Services.Redis {
		if paramType == "ip" {
			targetIP := net.ParseIP(entry)
			if targetIP == nil {
				return http.StatusBadRequest, errors.New("invalid ip address")
			}
		}

		if paramType == "cidr" {
			_, _, err := net.ParseCIDR(entry)
			if err != nil {
				return http.StatusBadRequest, errors.New("invalid cidr")
			}
		}
		ipCmd := services.RedisClient.SRem(ctx, "denylist:"+paramType+"s", entry)
		res, err := ipCmd.Result()
		log.Print(res)
		if err != nil {
			return http.StatusInternalServerError, errors.New("failed to remove entry")
		}
		return http.StatusOK, nil
	} else {
		return http.StatusBadRequest, errors.New("no dynamic source configured")
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
	sourceListRaw, sourceListExists := raw["sourceList"]

	if !sourceListExists {
		return util.NamedRiskHandler{}, errors.New("denylist: must provide source")
	}

	sourceList, ok := sourceListRaw.(string)
	if !ok {
		return util.NamedRiskHandler{}, errors.New("denylist: source list configuration must be a string")
	}

	if sourceList != "static" && sourceList != util.Services.Redis {
		return util.NamedRiskHandler{}, errors.New("denylist: invalid source list")
	}

	denylistConfig.sourceList = sourceList

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

	// Once all parsers have passed, indicate the rule is properly configured
	denylistConfig.configured = true
	denylistConfig.cidrs = cidrs
	denylistConfig.ips = ips

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
