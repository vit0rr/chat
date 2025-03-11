package deps

import (
	"github.com/vit0rr/chat/config"
	"go.mongodb.org/mongo-driver/mongo"
)

type Deps struct {
	Config config.Config
	Mongo  *mongo.Database
}

func New(config config.Config, db *mongo.Database) *Deps {
	return &Deps{
		Config: config,
		Mongo:  db,
	}
}
