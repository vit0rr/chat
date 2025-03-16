package config

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

// Config is the top-level config
type Config struct {
	Server Server `hcl:"server,block"`
	API    API    `hcl:"api,block"`
	Env    Env    `hcl:"env,block"`
	JWT    struct {
		Secret string `env:"JWT_SECRET" envDefault:"secret-key"`
	}
}

type Env struct {
	Port string `hcl:"port,attr"`
	Host string `hcl:"host,attr"`
	Env  string `hcl:"env,attr"`
	AllowedOrigins string `hcl:"allowed_origins,attr"`
}

// Server related config
type Server struct {
	BindAddr   string `hcl:"bind_addr,attr"`
	LogLevel   string `hcl:"log_level,attr"`
	CtxTimeout int    `hcl:"ctx_timeout,attr"`
}

// GetConfig returns a config from an hcl file
func GetConfig(path string) (Config, error) {
	config := Config{}
	err := hclsimple.DecodeFile(path, nil, &config)

	return config, err
}

// DefaultConfig returns a default config
func DefaultConfig(cfg Config) Config {

	return Config{
		Server: Server{
			BindAddr:   fmt.Sprintf(":%s", os.Getenv("PORT")),
			LogLevel:   "INFO",
			CtxTimeout: 5,
		},
		API: GetDefaltAPIConfig(cfg),
		Env: Env{
			Port: os.Getenv("PORT"),
			Host: os.Getenv("HOST"),
			Env:  os.Getenv("ENV"),
			AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
		},
	}
}
