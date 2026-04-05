package store

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
	"zorvyn/internal/models"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrEmailExists = errors.New("email already exists")
)

type Store struct {
	mu            sync.RWMutex
	users         map[string]*models.User
	records       map[string]*models.FinancialRecord
	blockedTokens map[string]struct{}
}

func New() *Store {
	return &Store{
		users:         make(map[string]*models.User),
		records:       make(map[string]*models.FinancialRecord),
		blockedTokens: make(map[string]struct{}),
	}
}

// ── Token Blocklist ──────────────────────────────────────────────────────────

func (s *Store) BlockToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blockedTokens[token] = struct{}{}
}

func (s *Store) IsTokenBlocked(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, blocked := s.blockedTokens[token]
	return blocked
}

// ── Users ────────────────────────────────────────────────────────────────────

func (s *Store) CreateUser(u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.users {
		if strings.EqualFold(existing.Email, u.Email) {
			return ErrEmailExists
		}
	}
	s.users[u.ID] = u
	return nil
}

func (s *Store) GetUserByID(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) ListUsers() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *Store) UpdateUser(u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[u.ID]; !ok {
		return ErrNotFound
	}
	s.users[u.ID] = u
	return nil
}

func (s *Store) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return ErrNotFound
	}
	delete(s.users, id)
	return nil
}

// ── Financial Records ────────────────────────────────────────────────────────

func (s *Store) CreateRecord(r *models.FinancialRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[r.ID] = r
	return nil
}

func (s *Store) GetRecordByID(id string) (*models.FinancialRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	if !ok || r.Deleted {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *Store) ListRecords(f models.RecordFilter) ([]*models.FinancialRecord, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	search := strings.ToLower(f.Search)
	var filtered []*models.FinancialRecord
	for _, r := range s.records {
		if r.Deleted {
			continue
		}
		if f.Type != "" && r.Type != f.Type {
			continue
		}
		if f.Category != "" && !strings.EqualFold(r.Category, f.Category) {
			continue
		}
		if f.From != nil && r.Date.Before(*f.From) {
			continue
		}
		if f.To != nil && r.Date.After(*f.To) {
			continue
		}
		if search != "" {
			haystack := strings.ToLower(r.Category + " " + r.Description)
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, r)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Date.After(filtered[j].Date)
	})

	total := len(filtered)
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	start := (f.Page - 1) * f.Limit
	if start >= total {
		return []*models.FinancialRecord{}, total
	}
	end := start + f.Limit
	if end > total {
		end = total
	}
	return filtered[start:end], total
}

func (s *Store) UpdateRecord(r *models.FinancialRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.records[r.ID]
	if !ok || existing.Deleted {
		return ErrNotFound
	}
	s.records[r.ID] = r
	return nil
}

func (s *Store) SoftDeleteRecord(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[id]
	if !ok || r.Deleted {
		return ErrNotFound
	}
	r.Deleted = true
	r.UpdatedAt = time.Now()
	return nil
}

func (s *Store) AllActiveRecords() []*models.FinancialRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*models.FinancialRecord
	for _, r := range s.records {
		if !r.Deleted {
			out = append(out, r)
		}
	}
	return out
}
