package rules

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"rba/services/database"
	"rba/util"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type DenyListSource interface {
	AddNetwork(ctx context.Context, cidr string) error
	RemoveNetwork(ctx context.Context, ip string) error
	Contains(ctx context.Context, ip string) (bool, error)
	GetNetworks(ctx context.Context) ([]string, error)
}

type StaticSource struct {
	networks []netip.Prefix
}

var Denylist DenyListSource

func (s *StaticSource) AddNetwork(ctx context.Context, network string) error {
	var parsedNetwork netip.Prefix
	p, err := netip.ParsePrefix(network)
	if err == nil {
		parsedNetwork = p
	} else {
		addr, err := netip.ParseAddr(network)
		if err != nil {
			return util.ErrInvalidNetwork
		}
		parsedNetwork = netip.PrefixFrom(addr, addr.BitLen())
	}
	for i := range s.networks {
		if s.networks[i] == parsedNetwork {
			return util.ErrNetworkAlreadyExists
		}
	}
	s.networks = append(s.networks, parsedNetwork)
	return nil
}

func (s *StaticSource) RemoveNetwork(ctx context.Context, network string) error {
	var parsedNetwork netip.Prefix
	addr, err := netip.ParseAddr(network)
	if err == nil {
		parsedNetwork = netip.PrefixFrom(addr, addr.BitLen())
	}
	p, err := netip.ParsePrefix(parsedNetwork.String())
	if err != nil {
		return util.ErrInvalidNetwork
	}

	for i := range s.networks {
		if s.networks[i] == p {
			s.networks = append(s.networks[:i], s.networks[i+1:]...)
		}
	}
	return nil
}

func (s *StaticSource) Contains(ctx context.Context, ip string) (bool, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false, util.ErrInvalidNetwork
	}

	for _, p := range s.networks {
		if p.Contains(addr) {
			return true, nil
		}
	}
	return false, nil
}

func (s *StaticSource) GetNetworks(ctx context.Context) ([]string, error) {
	stringNetworks := make([]string, 0, len(s.networks))
	for _, network := range s.networks {
		stringNetworks = append(stringNetworks, network.String())
	}
	return stringNetworks, nil
}

type DBSource struct {
	networks []netip.Prefix
}

func (s *DBSource) AddNetwork(ctx context.Context, cidr string) error {
	db, err := database.GetDB()
	if err != nil {
		return util.ErrUnknown
	}
	return db.AddDenylistNetwork(ctx, cidr)
}

func (s *DBSource) RemoveNetwork(ctx context.Context, cidr string) error {
	db, err := database.GetDB()
	if err != nil {
		return util.ErrUnknown
	}
	return db.RemoveDenylistNetwork(ctx, cidr)
}

func (s *DBSource) Contains(ctx context.Context, ip string) (bool, error) {
	db, err := database.GetDB()
	if err != nil {
		return false, util.ErrUnknown
	}
	exists, err := db.IPDenied(ctx, ip)
	return exists, nil
}

func (s *DBSource) GetNetworks(ctx context.Context) ([]string, error) {
	db, err := database.GetDB()
	if err != nil {
		return nil, util.ErrUnknown
	}
	networks, err := db.GetDenylistNetworks(ctx)
	return networks, nil
}

type DenyListConfig struct {
	SourceList string   `yaml:"sourceList" validate:"required,oneof=static database"`
	IPs        []string `yaml:"ips" validate:"omitempty,dive,ip"`
	CIDRs      []string `yaml:"cidrs" validate:"omitempty,dive,cidr"`
	Strategy   string   `yaml:"strategy" validate:"required,oneof=override average"`
}

func parseDenylistRule(raw yaml.Node) (util.NamedRiskHandler, error) {
	var cfg DenyListConfig
	if err := raw.Decode(&cfg); err != nil {
		return util.NamedRiskHandler{}, fmt.Errorf("failed to decode YAML: %w", err)
	}

	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return util.NamedRiskHandler{}, fmt.Errorf("error parsing denylist configuration: %w", err)
	}

	if cfg.SourceList == "static" {
		Denylist = &StaticSource{}
	}
	if cfg.SourceList == "database" {
		Denylist = &DBSource{}
	}
	for _, network := range append(cfg.IPs, cfg.CIDRs...) {
		Denylist.AddNetwork(context.Background(), network)
	}

	return util.NamedRiskHandler{
		Name:     util.Rules.Denylist,
		Strategy: cfg.Strategy,
		Handler: func(ctx context.Context, args map[string]interface{}) util.RiskResult {
			base := util.RiskResult{
				Name:     util.Rules.Denylist,
				Strategy: cfg.Strategy,
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

			ipDenied, err := Denylist.Contains(ctx, ip)
			if err != nil {
				errText := "failed to check containment"
				log.Printf("failed to check denylist containment: %s", ip)
				result := base
				result.Err = &errText
				return result
			}

			result := base
			if ipDenied {
				result.Score = 1
			}
			return result
		},
	}, nil
}
