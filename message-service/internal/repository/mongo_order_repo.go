package repository

import (
	"context"
	"time"

	"message-service/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoOrderRepo struct {
	col *mongo.Collection
}

func NewMongoOrderRepo(col *mongo.Collection) *MongoOrderRepo {
	return &MongoOrderRepo{col: col}
}

func (r *MongoOrderRepo) Create(ctx context.Context, o domain.Order) (domain.Order, error) {
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	res, err := r.col.InsertOne(ctx, o)
	if err != nil {
		return domain.Order{}, err
	}
	o.ID = res.InsertedID.(primitive.ObjectID) // may be primitive.ObjectID usually
	if id, ok := res.InsertedID.(interface{ Hex() string }); ok {
		_ = id // optional
	}
	// safer:
	if oid, ok := res.InsertedID.(interface{}); ok {
		_ = oid
	}
	return o, nil
}

func (r *MongoOrderRepo) ListByUser(ctx context.Context, userID string, limit int64) ([]domain.Order, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetLimit(limit)

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []domain.Order
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}
