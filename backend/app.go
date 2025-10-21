package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

var db *sql.DB
var jwtSecret []byte

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize database
	initDB()
	defer db.Close()

	// Set JWT secret
	jwtSecret = []byte(getEnv("JWT_SECRET", "your-secret-key"))

	// Initialize router
	r := mux.NewRouter()

	// Auth routes
	r.HandleFunc("/api/register", registerHandler).Methods("POST")
	r.HandleFunc("/api/login", loginHandler).Methods("POST")
	r.HandleFunc("/api/profile", authenticateMiddleware(profileHandler)).Methods("GET")

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func initDB() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "loginapp")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Successfully connected to database")
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	var existingUser User
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&existingUser.ID)
	if err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Insert user
	var user User
	err = db.QueryRow(`
		INSERT INTO users (email, password_hash) 
		VALUES ($1, $2) 
		RETURNING id, email, created_at, updated_at`,
		req.Email, string(hashedPassword),
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, err := generateJWT(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find user by email
	var user User
	err := db.QueryRow(`
		SELECT id, email, password_hash, created_at, updated_at 
		FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := generateJWT(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	var user User
	err := db.QueryRow(`
		SELECT id, email, created_at, updated_at 
		FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func generateJWT(userID int, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func authenticateMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(bearerToken[1], &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// Add user info to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "userID", claims.UserID)
			ctx = context.WithValue(ctx, "email", claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
