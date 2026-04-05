package handlers

import (
	"net/http"
	"strings"
	"time"
	"zorvyn/internal/middleware"
	"zorvyn/internal/models"
	"zorvyn/internal/store"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	store *store.Store
}

func NewUserHandler(s *store.Store) *UserHandler {
	return &UserHandler{store: s}
}

// ListUsers godoc
// @Summary      List all users
// @Description  Returns all registered users sorted by creation date. Admin only.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   models.User
// @Failure      401  {object}  models.ErrorResponse
// @Failure      403  {object}  models.ErrorResponse
// @Router       /users [get]
func (h *UserHandler) ListUsers(c echo.Context) error {
	return c.JSON(http.StatusOK, h.store.ListUsers())
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Returns a single user by their UUID. Admin only.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  models.User
// @Failure      401  {object}  models.ErrorResponse
// @Failure      403  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /users/{id} [get]
func (h *UserHandler) GetUser(c echo.Context) error {
	user, err := h.store.GetUserByID(c.Param("id"))
	if err != nil {
		return httpErr(http.StatusNotFound, "NOT_FOUND", "user not found")
	}
	return c.JSON(http.StatusOK, user)
}

// CreateUser godoc
// @Summary      Create user
// @Description  Creates a new user with a hashed password and assigns a role. Admin only.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      models.CreateUserRequest  true  "New user details"
// @Success      201   {object}  models.User
// @Failure      400   {object}  models.ErrorResponse
// @Failure      401   {object}  models.ErrorResponse
// @Failure      403   {object}  models.ErrorResponse
// @Failure      409   {object}  models.ErrorResponse
// @Router       /users [post]
func (h *UserHandler) CreateUser(c echo.Context) error {
	var req models.CreateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Name == "" {
		return httpErr(http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return httpErr(http.StatusInternalServerError, "HASH_ERROR", "could not process password")
	}
	user := &models.User{
		ID:           uuid.NewString(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		Active:       true,
		CreatedAt:    time.Now(),
	}
	if err := h.store.CreateUser(user); err != nil {
		if err == store.ErrEmailExists {
			return httpErr(http.StatusConflict, "EMAIL_EXISTS", "a user with this email already exists")
		}
		return httpErr(http.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return c.JSON(http.StatusCreated, user)
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Updates a user's name, role, or active status. All fields are optional. Admin only.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                    true  "User UUID"
// @Param        body  body      models.UpdateUserRequest  true  "Fields to update"
// @Success      200   {object}  models.User
// @Failure      400   {object}  models.ErrorResponse
// @Failure      401   {object}  models.ErrorResponse
// @Failure      403   {object}  models.ErrorResponse
// @Failure      404   {object}  models.ErrorResponse
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c echo.Context) error {
	user, err := h.store.GetUserByID(c.Param("id"))
	if err != nil {
		return httpErr(http.StatusNotFound, "NOT_FOUND", "user not found")
	}
	var req models.UpdateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	if req.Name != "" {
		name := strings.TrimSpace(req.Name)
		if name == "" {
			return httpErr(http.StatusBadRequest, "VALIDATION_ERROR", "name cannot be empty")
		}
		user.Name = name
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Active != nil {
		user.Active = *req.Active
	}
	if err := h.store.UpdateUser(user); err != nil {
		return httpErr(http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary      Delete user
// @Description  Permanently removes a user. An admin cannot delete their own account. Admin only.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User UUID"
// @Success      204  "No Content"
// @Failure      401  {object}  models.ErrorResponse
// @Failure      403  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c echo.Context) error {
	caller := middleware.CurrentUser(c)
	if caller.ID == c.Param("id") {
		return httpErr(http.StatusForbidden, "SELF_DELETE", "you cannot delete your own account")
	}
	if err := h.store.DeleteUser(c.Param("id")); err != nil {
		return httpErr(http.StatusNotFound, "NOT_FOUND", "user not found")
	}
	return c.NoContent(http.StatusNoContent)
}
