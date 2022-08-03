package u

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoCollectionMustExist return true if collection exists.
func MongoCollectionMustExist(mongoDB *mongo.Database, name string) bool {
	names, err := mongoDB.ListCollectionNames(context.Background(), bson.M{"name": name})
	if err != nil {
		Panic(ErrMongoQueryErr(err))
	}
	return len(names) > 0
}

// MustSetupMongoCollection creates collection if not exists.
func MustSetupMongoCollection(mongoDB *mongo.Database, name string, validator bson.M, indexes []mongo.IndexModel) {
	c := mongoDB.Collection(name, nil)
	if MongoCollectionMustExist(mongoDB, name) {
		// update validator
		result := mongoDB.RunCommand(context.Background(), bson.D{
			{"collMod", name},
			{"validator", validator},
		})
		if result.Err() != nil {
			Panic(ErrMongoWriteErr(result.Err()))
		}

		// create indexes is an idempotent operation.
		if len(indexes) > 0 {
			iv := c.Indexes()
			_, err := iv.CreateMany(context.Background(), indexes)
			if err != nil {
				Panic(ErrMongoWriteErr(err))
			}
		}
		return
	}

	// create collection and validator
	opts := options.CreateCollection()
	opts.Validator = validator
	err := mongoDB.CreateCollection(context.Background(), name, opts)
	if err != nil {
		Panic(ErrMongoWriteErr(err))
	}

	// create indexes
	if len(indexes) > 0 {
		iv := c.Indexes()
		_, err = iv.CreateMany(context.Background(), indexes)
		if err != nil {
			Panic(ErrMongoWriteErr(err))
		}
	}
}
