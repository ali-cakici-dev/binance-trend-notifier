package pkg

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	DB     *mongo.Database
}

type Config struct {
	ConnectionURI string
	DatabaseName  string // instead we can use the database name from the connection URI
}

func NewMongoClient(config Config) (*MongoInstance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.ConnectionURI))
	if err != nil {
		return nil, err
	}

	return &MongoInstance{
		Client: client,
		DB:     client.Database(config.DatabaseName),
	}, nil
}

func (mi *MongoInstance) Close() {
	if mi.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := mi.Client.Disconnect(ctx)
		if err != nil {
			return
		}
	}
}
