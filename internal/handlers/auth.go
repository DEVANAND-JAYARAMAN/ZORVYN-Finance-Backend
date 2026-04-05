package handlers

import (
	"net/http"
	"zorvyn/internal/middleware"
	"zorvyn/internal/models"
	"zorvyn/internal/store"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	store *store.Store
}

func NewAuthHandler(s *store.Store) *AuthHandler {
	return &AuthHandler{store: s}
}

// Login godoc
// @Summary      Login
// @Description  Authenticate with email and password. Returns a JWT Bearer token valid for 24 hours.
// @Description
// @Description  **Dummy credentials:**
// @Description  - `admin@zorvyn.io` / `Admin@123` → role: admin
// @Description  - `analyst@zorvyn.io` / `Analyst@123` → role: analyst
// @Description  - `viewer@zorvyn.io` / `Viewer@123` → role: viewer
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      models.LoginRequest  true  "Email and password"
// @Success      200   {object}  models.LoginResponse
// @Failure      400   {object}  models.ErrorResponse
// @Failure      401   {object}  models.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req models.LoginRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil || !user.Active {
		return httpErr(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return httpErr(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
	}
	token, err := middleware.GenerateToken(user)
	if err != nil {
		return httpErr(http.StatusInternalServerError, "TOKEN_ERROR", "could not generate token")
	}
	return c.JSON(http.StatusOK, models.LoginResponse{Token: token, User: *user})
}

// Logout godoc
// @Summary      Logout
// @Description  Revokes the current JWT token. The token will be rejected on subsequent requests.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  models.ErrorResponse
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	token := middleware.CurrentToken(c)
	h.store.BlockToken(token)
	return c.JSON(http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// Me godoc
// @Summary      Current user
// @Description  Returns the profile of the currently authenticated user.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.User
// @Failure      401  {object}  models.ErrorResponse
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c echo.Context) error {
	return c.JSON(http.StatusOK, middleware.CurrentUser(c))
}
