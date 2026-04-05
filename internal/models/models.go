package models

import "time"

type Role string

const (
	RoleViewer  Role = "viewer"
	RoleAnalyst Role = "analyst"
	RoleAdmin   Role = "admin"
)

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

type RecordType string

const (
	RecordIncome  RecordType = "income"
	RecordExpense RecordType = "expense"
)

type FinancialRecord struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Amount      float64    `json:"amount"`
	Type        RecordType `json:"type"`
	Category    string     `json:"category"`
	Date        time.Time  `json:"date"`
	Description string     `json:"description"`
	Deleted     bool       `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ── Request / Response DTOs ──────────────────────────────────────────────────

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateUserRequest struct {
	Name     string `json:"name"     validate:"required"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     Role   `json:"role"     validate:"required,oneof=viewer analyst admin"`
}

type UpdateUserRequest struct {
	Name   string `json:"name"`
	Role   Role   `json:"role"   validate:"omitempty,oneof=viewer analyst admin"`
	Active *bool  `json:"active"`
}

type CreateRecordRequest struct {
	Amount      float64    `json:"amount"      validate:"required,gt=0"`
	Type        RecordType `json:"type"        validate:"required,oneof=income expense"`
	Category    string     `json:"category"    validate:"required"`
	Date        time.Time  `json:"date"        validate:"required"`
	Description string     `json:"description"`
}

type UpdateRecordRequest struct {
	Amount      *float64    `json:"amount"      validate:"omitempty,gt=0"`
	Type        *RecordType `json:"type"       validate:"omitempty,oneof=income expense"`
	Category    *string     `json:"category"`
	Date        *time.Time  `json:"date"`
	Description *string     `json:"description"`
}

type RecordFilter struct {
	Type     RecordType `query:"type"`
	Category string     `query:"category"`
	Search   string     `query:"search"`
	From     *time.Time `query:"from"`
	To       *time.Time `query:"to"`
	Page     int        `query:"page"`
	Limit    int        `query:"limit"`
}

type PaginatedRecords struct {
	Data  []*FinancialRecord `json:"data"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Pages int                `json:"pages"`
}

type DashboardSummary struct {
	TotalIncome       float64            `json:"total_income"`
	TotalExpenses     float64            `json:"total_expenses"`
	NetBalance        float64            `json:"net_balance"`
	ByCategoryIncome  map[string]float64 `json:"by_category_income"`
	ByCategoryExpense map[string]float64 `json:"by_category_expense"`
	RecentRecords     []FinancialRecord  `json:"recent_records"`
	MonthlyTrends     []MonthlyTrend     `json:"monthly_trends"`
}

type WeeklyTrend struct {
	Week    string  `json:"week"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

type MonthlyTrend struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

// ErrorResponse is the standard error envelope returned by all endpoints.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
