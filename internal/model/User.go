package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID       primitive.ObjectID `bson:"_id, omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Password string             `bson:"password" json:"password"`
	Email    string             `bson:"email" json:"email"`
	Credit   float64            `bson:"credit" json:"credit"`
}
