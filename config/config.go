package config

import (
	"flag"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// Global

// var (
// 	port = flag.Int("port", 3000, "The server port")
// 	addr = flag.String("address", "localhost", "The server address")
// )

var (
	local = flag.String("local", "../local.yaml", "Local config relative path")
)

// Config holds addresses, ports and db URIs
type Config struct {
	Server struct {
        Port int `yaml:"port"`
        Addr string `yaml:"address"`
    } `yaml:"server"`
}

// func New() *Config {
// 	return &Config{}
// }

func Load() *Config {
	flag.Parse()
	f, err := os.Open(*local)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer f.Close()

	var cfg Config

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}

	return &cfg
}
