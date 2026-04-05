package store

import (
	"testing"
	"time"
	"zorvyn/internal/models"
)

func TestCreateUserRejectsCaseInsensitiveDuplicateEmail(t *testing.T) {
	s := New()

	err := s.CreateUser(&models.User{
		ID:        "u1",
		Name:      "User One",
		Email:     "Admin@Zorvyn.io",
		Role:      models.RoleAdmin,
		Active:    true,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected create user error: %v", err)
	}

	err = s.CreateUser(&models.User{
		ID:        "u2",
		Name:      "User Two",
		Email:     "admin@zorvyn.io",
		Role:      models.RoleViewer,
		Active:    true,
		CreatedAt: time.Now(),
	})
	if err != ErrEmailExists {
		t.Fatalf("expected ErrEmailExists, got: %v", err)
	}
}
