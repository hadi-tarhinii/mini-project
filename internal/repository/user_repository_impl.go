package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mini-project/internal/model"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepositoryImpl struct {
	collection  *mongo.Collection
	redisClient *redis.Client
}

func NewUserRepositoryImpl(col *mongo.Collection, rdb *redis.Client) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		collection:  col,
		redisClient: rdb,
	}
}

func (r *UserRepositoryImpl) CreateUser(ctx context.Context, user *model.User) error {
	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepositoryImpl) DeleteUser(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("no user found with this id")
	}
	r.ClearCache(ctx, id)
	return nil
}

// Credit
func (r *UserRepositoryImpl) AddCredit(ctx context.Context, id string, amount float64) error {
	
	// convert string ID to mongo objID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %v", err)
	}

	//get the struct we are looking for
	filter := bson.M{"_id": objID}

	//do the math: $inc tells mongo to add this amount
	update := bson.M{
		"$inc": bson.M{"credit": amount},
	}

	//execute the update
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	//check if user is found and update
	if result.MatchedCount == 0 {
		return errors.New("User not found")
	}

	//clear cache if everything went right
	r.ClearCache(ctx, id)

	return nil

}

func (r *UserRepositoryImpl) DeductCredit(ctx context.Context, id string, amount float64) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %v", err)
	}

	filter := bson.M{"_id": objID}

	update := bson.M{
		"$inc": bson.M{"credit": -amount},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("User not found")
	}

	r.ClearCache(ctx, id)

	return nil
}

func (r *UserRepositoryImpl) Transfer(ctx context.Context, senderId string, receiverId string, amount float64) error {
    // 1. Start a Session for Atomicity
    session, err := r.collection.Database().Client().StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    // 2. Wrap everything in a Transaction
    _, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
        objSender, _ := primitive.ObjectIDFromHex(senderId)
        objReceiver, _ := primitive.ObjectIDFromHex(receiverId)

        // Step A: Deduct with balance check
        res, err := r.collection.UpdateOne(sessCtx, 
            bson.M{"_id": objSender, "credit": bson.M{"$gte": amount}}, 
            bson.M{"$inc": bson.M{"credit": -amount}},
        )
        if err != nil || res.MatchedCount == 0 {
            return nil, errors.New("insufficient funds or sender not found")
        }

        // Step B: Add to receiver
        res, err = r.collection.UpdateOne(sessCtx, 
            bson.M{"_id": objReceiver}, 
            bson.M{"$inc": bson.M{"credit": amount}},
        )
        if err != nil || res.MatchedCount == 0 {
            // This triggers a ROLLBACK - sender gets their money back!
            return nil, errors.New("receiver not found")
        }

        return nil, nil
    })

    if err == nil {
        r.ClearCache(ctx, senderId)
        r.ClearCache(ctx, receiverId)
    }
    return err
}
// Read
func (r *UserRepositoryImpl) GetAll(ctx context.Context) ([]model.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []model.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepositoryImpl) GetUserByID(ctx context.Context, id string) (model.User, error) {
	key := "user:" + id
	var user model.User

	data, err := r.redisClient.HGetAll(ctx, key).Result()

	if err == nil && len(data) > 0 {
		log.Printf("Redis Hit, user ID: %s", id)
		objID, _ := primitive.ObjectIDFromHex(data["id"])
		balance, _ := strconv.ParseFloat(data["credit"], 64)

		user = model.User{
			ID:       objID,
			Username: data["username"],
			Email:    data["email"],
			Credit:   balance,
		}
		return user, nil
	}
	log.Printf("Redis miss, fetching %s from mongoDB", id)

	ObjID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user, fmt.Errorf("invalid ID, format: %v", err)
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": ObjID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return user, errors.New("User not found")
		}
		return user, err
	}
	err = r.redisClient.HSet(ctx, key, map[string]interface{}{
		"id": user.ID.Hex(),
		"username": user.Username,
		"email": user.Email,
		"credit": user.Credit,
	}).Err()

	if err != nil {
		log.Printf("Failed to cache")
	} else {
		r.redisClient.Expire(ctx, key, 10*time.Minute)
	}
	return user, nil
}

func (r *UserRepositoryImpl) GetCreditByID(ctx context.Context, id string) (float64, error) {
	user, err := r.GetUserByID(ctx, id)
	if err != nil {
		return 0, err
	}
	return float64(user.Credit), nil
}

// Clear cache
func (r *UserRepositoryImpl) ClearCache(ctx context.Context, id string){
	key := "user:" + id
	err := r.redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to clear cache for user %s, %v", id, err)
	}
}