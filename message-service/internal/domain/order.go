package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Order struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	OrderID   string             `json:"order_id" bson:"order_id"`
	UserID    string             `json:"user_id" bson:"user_id"`
	Items     []string           `json:"items" bson:"items"`
	Amount    float64            `json:"amount" bson:"amount"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}
