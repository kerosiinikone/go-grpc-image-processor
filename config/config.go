package config

// Config holds addresses, ports and db URIs
type Config struct{}

func New() *Config {
	return &Config{}
}
