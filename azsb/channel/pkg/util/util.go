package util

import (
	"os"
)

const (
	defaultMaxIdleConnections        = 1000
	defaultMaxIdleConnectionsPerHost = 100

	clientID = "az-servicebus-ch-dispatcher"
)

// AzsbConfig info for Azure Service Bus connection
type AzsbConfig struct {
	ClientID            string
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	ConnectionString    string
}

// GetAzsbConfig returns a AzsbConfig data
func GetAzsbConfig() AzsbConfig {
	return AzsbConfig{
		ClientID:            clientID,
		MaxIdleConns:        getMaxIdleConnections(),
		MaxIdleConnsPerHost: getMaxIdleConnectionsPerHost(),
		ConnectionString:    getConnectionString(),
	}
}

func getConnectionString() string {
	return getEnv("SB_CONNECTION", "")
}

// getMaxIdleConnections returns the max number of idle connections
func getMaxIdleConnections() int {
	return defaultMaxIdleConnections
}

// getMaxIdleConnections returns the max number of idle connections per host
func getMaxIdleConnectionsPerHost() int {
	return defaultMaxIdleConnectionsPerHost
}

func getEnv(envKey string, fallback string) string {
	val, ok := os.LookupEnv(envKey)
	if !ok {
		return fallback
	}
	return val
}
