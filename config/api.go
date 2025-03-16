package config

import (
	"os"
)

type API struct {
	Mongo Mongo `hcl:"mongo,block"`
	Redis Redis `hcl:"redis,block"`
	BaseURL BaseURL `hcl:"base_url,block"`
}

type Mongo struct {
	Dsn string `hcl:"dsn,attr"`
}

type Redis struct {
	Dsn string `hcl:"dsn,attr"`
}

type BaseURL struct {
	Url string `hcl:"url,attr"`
}

type BackendURL struct {
	Url string `hcl:"url,attr"`
}

func GetDefaltAPIConfig(cfg Config) API {
	return API{
		Mongo: Mongo{
			Dsn: os.Getenv("DATABASE_URL"),
		},
		Redis: Redis{
			Dsn: os.Getenv("REDIS_URL"),
		},
		BaseURL: BaseURL{
			Url: os.Getenv("BASE_URL"),
		},
	}
}
