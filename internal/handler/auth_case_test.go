package handler

import (
	"net/http"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
)

// seedAdmin creates the administrator account so registration is open
// (Register refuses while setup is pending).
func seedAdmin(t *testing.T) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("admin-password-123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := database.DB.Create(&model.User{
		Username: "root", Email: "root@example.com", DisplayName: "root",
		Password: string(hash), Role: "admin", Group: "default", Quota: -1, Status: "active",
	}).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}
}

// Usernames and emails are matched case-insensitively across registration
// and every login path; these tests pin that contract.
func TestUsernameCaseInsensitivity(t *testing.T) {
	h := setupCodeTestDB(t)
	seedAdmin(t)

	// Register the canonical account.
	rec, _ := postJSON(t, h.Login, "/api/auth/login", map[string]string{
		"account": "nobody", "password": "password123",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("precondition login should fail, got %d", rec.Code)
	}

	rec, _ = postJSON(t, h.Register, "/api/auth/register", map[string]string{
		"username": "Alice", "email": "Alice@Example.COM", "password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register Alice = %d: %s", rec.Code, rec.Body.String())
	}

	// Same name, different case → conflict (same identity).
	rec, _ = postJSON(t, h.Register, "/api/auth/register", map[string]string{
		"username": "alice", "email": "other@example.com", "password": "password123",
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("register alice (username case variant) = %d, want 409", rec.Code)
	}

	// Same email, different case → conflict.
	rec, _ = postJSON(t, h.Register, "/api/auth/register", map[string]string{
		"username": "alice", "email": "ALICE@example.com", "password": "password123",
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("register with case-variant email = %d, want 409", rec.Code)
	}

	// Login accepts case variants of the username.
	rec, _ = postJSON(t, h.Login, "/api/auth/login", map[string]string{
		"account": "ALICE", "password": "password123",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("login ALICE = %d: %s, want 200", rec.Code, rec.Body.String())
	}

	// Login accepts case variants of the email.
	rec, _ = postJSON(t, h.Login, "/api/auth/login", map[string]string{
		"account": "alice@example.com", "password": "password123",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("login by email = %d, want 200", rec.Code)
	}
}
