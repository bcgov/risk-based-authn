package rules

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net"
	"rba/services"
	"rba/util"
	"time"

	"github.com/redis/go-redis/v9"
)

type LoginData struct {
	IP        string  `json:"ip"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Timestamp int64   `json:"ts"`
	Accuracy  int64   `json:"acc"`
}

func storeLastLogin(ctx context.Context, username string, login LoginData, maxTravelTime time.Duration) error {
	key := util.Rules.ImpossibleTravel + ":" + username

	data, err := json.Marshal(login)
	if err != nil {
		return err
	}

	return services.RedisClient.Set(ctx, key, data, maxTravelTime).Err()
}

func getLastLogin(ctx context.Context, username string) (*LoginData, error) {
	key := util.Rules.ImpossibleTravel + ":" + username
	data, err := services.RedisClient.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // no previous login
	} else if err != nil {
		return nil, err
	}

	var login LoginData
	if err := json.Unmarshal(data, &login); err != nil {
		return nil, err
	}

	return &login, nil
}

func Geolocate(ip net.IP) (LoginData, error) {
	record, err := services.CityDB.City(ip)
	if err != nil {
		log.Println(err)
		return LoginData{}, err
	}

	return LoginData{
		IP:        ip.String(),
		Latitude:  record.Location.Latitude,
		Longitude: record.Location.Longitude,
		Timestamp: time.Now().Unix(),
		Accuracy:  int64(record.Location.AccuracyRadius),
	}, nil
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func parseImpossibleTravelRule(raw map[string]interface{}) (util.NamedRiskHandler, error) {
	if redisErr := services.PingRedis(); redisErr != nil {
		return util.NamedRiskHandler{}, errors.New(util.Rules.ImpossibleTravel + ": a valid redis connection is required for this rule. Check redis configuration")
	}

	if services.CityDB == nil {
		return util.NamedRiskHandler{}, errors.New(util.Rules.ImpossibleTravel + ": a valid geoIP connection is required for this rule. Check geoIP configuration")
	}

	speed, ok := raw["speedKilometersPerHour"].(int)

	if !ok {
		return util.NamedRiskHandler{}, errors.New(util.Rules.ImpossibleTravel + ": missing or invalid speedKilometersPerHour")
	}

	strategy, ok := raw["strategy"].(string)
	if !ok || !util.IsValidStrategy(strategy) {
		return util.NamedRiskHandler{}, errors.New(util.Rules.ImpossibleTravel + ": missing or invalid strategy")
	}

	earthCircumferenceKM := float64(40075)

	// Time required to circumnavigate the globe at the provided speed. Entries can be expired from redis at that point since all folow up locations are valid.
	magellanTime := time.Duration((earthCircumferenceKM / float64(speed)) * float64(time.Hour))

	return util.NamedRiskHandler{
		Name:     util.Rules.ImpossibleTravel,
		Strategy: strategy,
		Handler: func(ctx context.Context, args map[string]interface{}) util.RiskResult {

			base := util.RiskResult{
				Name:     util.Rules.ImpossibleTravel,
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

			account, err := util.GetStringField(args, "account")
			if err != nil {
				errText := "missing account"
				result := base
				result.Err = &errText
				return result
			}

			previousLogin, err := getLastLogin(ctx, account)
			if err != nil {
				errText := "failed to read login from redis"
				result := base
				result.Err = &errText
				return result
			}

			loginData, err := Geolocate(net.ParseIP(ip))
			if err != nil {
				errText := "failed to geolocate IP"
				result := base
				result.Err = &errText
				return result
			}

			if previousLogin == nil {
				storeLastLogin(ctx, account, loginData, magellanTime)
				return base
			}

			distance := haversine(previousLogin.Latitude, previousLogin.Longitude, loginData.Latitude, loginData.Longitude)

			timeDeltaHours := (float64(loginData.Timestamp) - float64(previousLogin.Timestamp)) / float64(3600)
			distanceDeltaKilometers := distance - float64(loginData.Accuracy) - float64(previousLogin.Accuracy)

			if float64(speed)*timeDeltaHours < float64(distanceDeltaKilometers) {
				base.Score = 1
			}

			storeLastLogin(ctx, account, loginData, magellanTime)

			result := base
			return result
		},
	}, nil
}
