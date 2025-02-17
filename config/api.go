package config

import (
	"os"
)

type API struct {
	Mongo Mongo `hcl:"mongo,block"`
	Redis Redis `hcl:"redis,block"`
}

type Mongo struct {
	Dsn string `hcl:"dsn,attr"`
}

type Redis struct {
	Dsn string `hcl:"dsn,attr"`
}

func GetDefaltAPIConfig(cfg Config) API {
	return API{
		Mongo: Mongo{
			Dsn: os.Getenv("DATABASE_URL"),
		},
		Redis: Redis{
			Dsn: os.Getenv("REDIS_URL"),
		},
	}
}
