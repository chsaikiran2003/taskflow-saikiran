package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email"`
	Password  string    `db:"password" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, map[string]string) {
	fields := map[string]string{}
	if req.Name == "" {
		fields["name"] = "is required"
	}
	if req.Email == "" {
		fields["email"] = "is required"
	}
	if len(req.Password) < 8 {
		fields["password"] = "must be at least 8 characters"
	}
	if len(fields) > 0 {
		return nil, fields
	}

	// Check email uniqueness
	var count int
	err := s.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users WHERE email=$1", req.Email)
	if err != nil {
		slog.Error("checking email uniqueness", "err", err)
		return nil, map[string]string{"email": "could not be validated"}
	}
	if count > 0 {
		return nil, map[string]string{"email": "already in use"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		slog.Error("hashing password", "err", err)
		return nil, map[string]string{"password": "could not be processed"}
	}

	user := &User{}
	err = s.db.QueryRowxContext(ctx,
		`INSERT INTO users (id, name, email, password) VALUES (gen_random_uuid(), $1, $2, $3)
		 RETURNING id, name, email, password, created_at`,
		req.Name, req.Email, string(hash),
	).StructScan(user)
	if err != nil {
		slog.Error("inserting user", "err", err)
		return nil, map[string]string{"email": "could not create user"}
	}

	token, err := generateToken(user)
	if err != nil {
		slog.Error("generating token", "err", err)
		return nil, map[string]string{"_": "could not generate token"}
	}

	return &AuthResponse{Token: token, User: *user}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, map[string]string) {
	fields := map[string]string{}
	if req.Email == "" {
		fields["email"] = "is required"
	}
	if req.Password == "" {
		fields["password"] = "is required"
	}
	if len(fields) > 0 {
		return nil, fields
	}

	user := &User{}
	err := s.db.GetContext(ctx, user, "SELECT * FROM users WHERE email=$1", req.Email)
	if err != nil {
		return nil, map[string]string{"email": "invalid credentials"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, map[string]string{"password": "invalid credentials"}
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, map[string]string{"_": "could not generate token"}
	}

	return &AuthResponse{Token: token, User: *user}, nil
}

func generateToken(user *User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET not set")
	}

	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}

func ValidateToken(tokenStr string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
