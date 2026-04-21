package service

import (
	"context"
	"errors"
	"fmt"
	"mini-project/internal/model"
	"mini-project/internal/repository"
	"os"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	// Authentication
	Login(ctx context.Context, username string, password string) (string, error)

	// Create
	CreateUser(ctx context.Context, user *model.User) error

	// Delete
	DeleteUser(ctx context.Context, id string) error

	// Credit Operations
	AddCredit(ctx context.Context, id string, amount float64) error
	DeductCredit(ctx context.Context, id string, amount float64) error
	Transfer(ctx context.Context, senderId string, receiverId string, amount float64) error

	// Read
	GetAll(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
	GetCreditByID(ctx context.Context, id string) (float64, error)
}

type UserServiceImpl struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &UserServiceImpl{
		userRepo: userRepo,
	}
}

// A secret key should ideally come from an Environment Variable
var jwtKey = []byte(os.Getenv("JWT_SECRET"))

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func (s *UserServiceImpl) generateToken(userID string) (string, error) {
	// 1. Set the expiration time (e.g., 24 hours)
	expirationTime := time.Now().Add(24 * time.Hour)

	// 2. Create the claims (the data inside the token)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// 3. Declare the token with the algorithm and claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 4. Create the JWT string using our secret key
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *UserServiceImpl) Login(ctx context.Context, username string, password string) (string, error){
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("wrong password")
	}

	return  s.generateToken(user.ID.Hex())
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, user *model.User) error {
	// Generate ObjectID if not set
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}

	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}
	user.Password = string(hashedPassword)

	return s.userRepo.CreateUser(ctx, user)
}

func (s *UserServiceImpl) DeleteUser(ctx context.Context, id string) error {
	return s.userRepo.DeleteUser(ctx, id)
}

func (s *UserServiceImpl) AddCredit(ctx context.Context, id string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	return s.userRepo.AddCredit(ctx, id, amount)
}

func (s *UserServiceImpl) DeductCredit(ctx context.Context, id string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	if user.Credit < amount {
		return errors.New("insufficient credit balance")
	}

	return s.userRepo.DeductCredit(ctx, id, amount)
}

func (s *UserServiceImpl) Transfer(ctx context.Context, senderId string, receiverId string, amount float64) error {
	if senderId == receiverId {
		return errors.New("Cannot transfer money to yourself")
	}
	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	sender, errS := s.userRepo.GetUserByID(ctx, senderId)
	if errS != nil {
		return errors.New("sender not found: " + errS.Error())
	}

	_, errR := s.userRepo.GetUserByID(ctx, receiverId)
	if errR != nil {
		return errors.New("receiver not found: " + errR.Error())
	}

	if sender.Credit < amount {
		return errors.New("insufficient funds: sender has " + fmt.Sprintf("%.2f", sender.Credit) + " but trying to transfer " + fmt.Sprintf("%.2f", amount))
	}

	return s.userRepo.Transfer(ctx, senderId, receiverId, amount)
}

func (s *UserServiceImpl) GetAll(ctx context.Context) ([]model.User, error) {
	return s.userRepo.GetAll(ctx)
}

func (s *UserServiceImpl) GetUserByID(ctx context.Context, id string) (model.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

func (s *UserServiceImpl) GetCreditByID(ctx context.Context, id string) (float64, error) {
	return s.userRepo.GetCreditByID(ctx, id)
}

func (s *UserServiceImpl) GetByUsername(ctx context.Context, username string)(model.User, error){
	return s.userRepo.GetByUsername(ctx, username)
}
