package config

import (
	"time"

	flag "github.com/spf13/pflag"
)

var flagServerAddress string
var flagLogLevel string
var flagDatabaseDSN string
var flagJWTSecret string
var flagTokenTTL time.Duration
var flagEnableHTTPS bool
var flagConfigPath string

func parseFlags() {
	if flag.Parsed() {
		return
	}

	flag.StringVarP(&flagServerAddress, "server-address", "s", "", "Server address")
	flag.StringVarP(&flagLogLevel, "log-level", "l", "info", "Log level")
	flag.StringVarP(&flagDatabaseDSN, "database-dsn", "d", "", "Database DSN")
	flag.StringVarP(&flagJWTSecret, "jwt-secret", "j", "", "JWT secret")
	flag.DurationVarP(&flagTokenTTL, "token-ttl", "t", 0, "Token TTL")
	flag.BoolVarP(&flagEnableHTTPS, "enable-https", "e", true, "Enable HTTPS")
	flag.StringVarP(&flagConfigPath, "config", "c", "", "Path to config file")

	flag.Parse()

}
