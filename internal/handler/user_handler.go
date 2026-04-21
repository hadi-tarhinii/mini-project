package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"mini-project/internal/middleware"
	"mini-project/internal/model"
	"mini-project/internal/service"
	"mini-project/internal/utils"

	"github.com/gorilla/mux"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// ============ LOGIN ============
// POST /login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	token, err := h.userService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"token": token,
	})
}

// ============ CREATE ============

// CreateUser creates a new user
// POST /users
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user model.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	if err := h.userService.CreateUser(ctx, &user); err != nil {
		log.Println("Error creating user:", err)
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User created successfully",
		"user":    user,
	})
}

// ============ READ ============

// GetAllUsers retrieves all users
// GET /users
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := h.userService.GetAll(ctx)
	if err != nil {
		log.Println("Error fetching users:", err)
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	utils.WriteJSON(w, http.StatusOK, users)
}

// GetUserByID retrieves a specific user by ID
// GET /users/{id}
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := r.Context()
	user, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		log.Println("Error fetching user:", err)
		utils.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, user)
}

// GetCreditByID retrieves credit balance for a user
// GET /users/{id}/credit
func (h *UserHandler) GetCreditByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := r.Context()
	credit, err := h.userService.GetCreditByID(ctx, id)
	if err != nil {
		log.Println("Error fetching credit:", err)
		utils.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]float64{"credit": credit})
}

// ============ DELETE ============

// DeleteUser deletes a user by ID
// DELETE /users/{id}
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := r.Context()
	if err := h.userService.DeleteUser(ctx, id); err != nil {
		log.Println("Error deleting user:", err)
		utils.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// ============ CREDIT OPERATIONS ============

// AddCredit adds credit to a user's account
// POST /users/{id}/credit/add
func (h *UserHandler) AddCredit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Amount float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	if err := h.userService.AddCredit(ctx, id, req.Amount); err != nil {
		log.Println("Error adding credit:", err)
		utils.WriteError(w, http.StatusInternalServerError, "Failed to add credit")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Credit added successfully",
		"amount":  req.Amount,
	})
}

// DeductCredit deducts credit from a user's account
// POST /users/{id}/credit/deduct
func (h *UserHandler) DeductCredit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Amount float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	if err := h.userService.DeductCredit(ctx, id, req.Amount); err != nil {
		log.Println("Error deducting credit:", err)
		utils.WriteError(w, http.StatusInternalServerError, "Failed to deduct credit")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Credit deducted successfully",
		"amount":  req.Amount,
	})
}

// Transfer credit from two users
// POST/users/{id}/transfer
func (h *UserHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	// Get the sender ID from the context (set by Middleware)
	senderId, ok := r.Context().Value("id").(string)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Use lowercase 'json' tags to match standard frontend naming
	var req struct {
		Amount     float64 `json:"amount"`
		ReceiverId string  `json:"ReceiverId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Safety: Don't allow transferring $0 or negative money
	if req.Amount <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "Amount must be greater than zero")
		return
	}

	ctx := r.Context()
	err := h.userService.Transfer(ctx, senderId, req.ReceiverId, req.Amount)

	if err != nil {
		// Log the real error for you to see in the terminal
		log.Printf("Transfer Error: %v", err)

		// Return a specific message to the user
		if err.Error() == "insufficient funds" || err.Error() == "receiver not found" {
			utils.WriteError(w, http.StatusBadRequest, err.Error())
		} else {
			utils.WriteError(w, http.StatusInternalServerError, "An internal error occurred during transfer")
		}
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Funds transferred successfully",
	})
}

// RegisterRoutes registers all user routes
func (h *UserHandler) RegisterRoutes(router *mux.Router) {
	// Public routes (no authentication required)
	router.HandleFunc("/login", h.Login).Methods(http.MethodPost)
	router.HandleFunc("/users", h.CreateUser).Methods(http.MethodPost)

	// Protected routes (authentication required)
	protected := router.NewRoute().Subrouter()
	protected.Use(middleware.AuthMiddleware)

	protected.HandleFunc("/users", h.GetAllUsers).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}", h.GetUserByID).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}/credit", h.GetCreditByID).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}", h.DeleteUser).Methods(http.MethodDelete)
	protected.HandleFunc("/users/{id}/credit/add", h.AddCredit).Methods(http.MethodPost)
	protected.HandleFunc("/users/{id}/credit/deduct", h.DeductCredit).Methods(http.MethodPost)
	protected.HandleFunc("/users/{id}/credit/transfer", h.Transfer).Methods(http.MethodPost)
}
