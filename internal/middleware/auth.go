package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"zorvyn/internal/models"
	"zorvyn/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

const jwtSecret = "zorvyn-secret-key-2025"

type Claims struct {
	UserID string      `json:"user_id"`
	Role   models.Role `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func apiErr(code, msg string) *echo.HTTPError {
	return &echo.HTTPError{
		Code:    http.StatusUnauthorized,
		Message: models.ErrorResponse{Code: code, Message: msg},
	}
}

func Auth(s *store.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				return &echo.HTTPError{
					Code:    http.StatusUnauthorized,
					Message: models.ErrorResponse{Code: "MISSING_TOKEN", Message: "authorization header required"},
				}
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			if s.IsTokenBlocked(tokenStr) {
				return &echo.HTTPError{
					Code:    http.StatusUnauthorized,
					Message: models.ErrorResponse{Code: "TOKEN_REVOKED", Message: "token has been revoked"},
				}
			}

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if t.Method != jwt.SigningMethodHS256 {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				return &echo.HTTPError{
					Code:    http.StatusUnauthorized,
					Message: models.ErrorResponse{Code: "INVALID_TOKEN", Message: "token is invalid or expired"},
				}
			}
			user, err := s.GetUserByID(claims.UserID)
			if err != nil || !user.Active {
				return &echo.HTTPError{
					Code:    http.StatusUnauthorized,
					Message: models.ErrorResponse{Code: "USER_INACTIVE", Message: "user not found or deactivated"},
				}
			}
			c.Set("user", user)
			c.Set("token", tokenStr)
			return next(c)
		}
	}
}

func RequireRole(roles ...models.Role) echo.MiddlewareFunc {
	allowed := make(map[models.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("user").(*models.User)
			if !ok {
				return &echo.HTTPError{
					Code:    http.StatusUnauthorized,
					Message: models.ErrorResponse{Code: "UNAUTHENTICATED", Message: "authentication required"},
				}
			}
			if _, permitted := allowed[user.Role]; !permitted {
				return &echo.HTTPError{
					Code:    http.StatusForbidden,
					Message: models.ErrorResponse{Code: "FORBIDDEN", Message: "your role does not have permission for this action"},
				}
			}
			return next(c)
		}
	}
}

// RateLimit returns a simple per-server rate limiter middleware.
func RateLimit(rps float64, burst int) echo.MiddlewareFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !limiter.Allow() {
				return &echo.HTTPError{
					Code:    http.StatusTooManyRequests,
					Message: models.ErrorResponse{Code: "RATE_LIMITED", Message: "too many requests, please slow down"},
				}
			}
			return next(c)
		}
	}
}

func CurrentUser(c echo.Context) *models.User {
	u, _ := c.Get("user").(*models.User)
	return u
}

func CurrentToken(c echo.Context) string {
	t, _ := c.Get("token").(string)
	return t
}

// suppress unused import warning
var _ = apiErr
