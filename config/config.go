package config

import "flag"

var (
	port = flag.Int("port", 3000, "The server port")
	addr = flag.String("address", "localhost", "The server address")
)

// Config holds addresses, ports and db URIs
type Config struct {
	Addr string
	Port int
}

func New() *Config {
	flag.Parse()
	
	return &Config{
		Addr: *addr,
		Port: *port,
	}
}
