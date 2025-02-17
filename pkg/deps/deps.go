package deps

import (
	"github.com/vit0rr/chat/config"
	"go.mongodb.org/mongo-driver/mongo"
)

type Deps struct {
	Config config.Config
	Mongo  *mongo.Client
}

func New(config config.Config, mongoClient *mongo.Client) *Deps {
	return &Deps{
		Config: config,
		Mongo:  mongoClient,
	}
}
