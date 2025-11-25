package rules
import "time"


type PasswordSprayRule struct {
    Name             string        `yaml:"name"`
    Timeframe        time.Duration `yaml:"timeframe"`
    AttemptsAllowed  int           `yaml:"attempts_allowed"`
    DistinctAccounts int           `yaml:"distinct_accounts"`
}

type LoginAttempt struct {
    Username  string
    IP        string
    Timestamp time.Time
    Success   bool
}

// Filter attempts for a given IP within the timeframe
func filterAttempts(attempts []LoginAttempt, ip string, cutoff time.Time) []LoginAttempt {
	var recent []LoginAttempt
	for _, a := range attempts {
		if a.IP == ip && !a.Success && a.Timestamp.After(cutoff){
			recent = append(recent, a)
		}
	}
	return recent
}

// Get unique usernames from attempts
func uniqueUsernames(attempts []LoginAttempt) []string {
	seen:= make(map[string]struct{})
	var users []string
	for _, a:= range attempts {
		if _, ok := seen[a.Username]; !ok {
			seen[a.Username] = struct {}{}
			users = append(users, a.Username)
		}
	}
	return users
}

// Detect password spray based on rule config
func detectPasswordSpray(ip string, attempts []LoginAttempt, rule PasswordSprayRule) bool {
	cutoff:= time.Now().Add(-rule.Timeframe)
	recent := filterAttempts(attempts, ip, cutoff)

	if len(recent) >= rule.AttemptsAllowed {
		accounts := uniqueUsernames(recent)
		if len(accounts) >= rule.DistinctAccounts {
			return true // suspicious
		}
	}
	return false
}