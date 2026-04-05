package main

import (
	"fmt"
	"net/http"
	"time"
	"zorvyn/internal/handlers"
	mw "zorvyn/internal/middleware"
	"zorvyn/internal/models"
	"zorvyn/internal/store"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"golang.org/x/crypto/bcrypt"

	_ "zorvyn/docs"
)

// ── Validator ────────────────────────────────────────────────────────────────

type echoValidator struct{ v *validator.Validate }

func (ev *echoValidator) Validate(i interface{}) error { return ev.v.Struct(i) }

// ── Custom Error Handler ─────────────────────────────────────────────────────

func customErrorHandler(err error, c echo.Context) {
	he, ok := err.(*echo.HTTPError)
	if !ok {
		he = &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: models.ErrorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"},
		}
	}
	// If message is already an ErrorResponse, send as-is
	if _, isStructured := he.Message.(models.ErrorResponse); !isStructured {
		he.Message = models.ErrorResponse{Code: "ERROR", Message: fmt.Sprintf("%v", he.Message)}
	}
	if !c.Response().Committed {
		_ = c.JSON(he.Code, he.Message)
	}
}

// ── Seed Data ────────────────────────────────────────────────────────────────

func seedData(s *store.Store) {
	type seedUser struct {
		name, email, password string
		role                  models.Role
	}
	users := []seedUser{
		{"Alice Admin", "admin@zorvyn.io", "Admin@123", models.RoleAdmin},
		{"Bob Analyst", "analyst@zorvyn.io", "Analyst@123", models.RoleAnalyst},
		{"Carol Viewer", "viewer@zorvyn.io", "Viewer@123", models.RoleViewer},
	}

	userIDs := make([]string, 0, len(users))
	for _, u := range users {
		hash, _ := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		user := &models.User{
			ID:           uuid.NewString(),
			Name:         u.name,
			Email:        u.email,
			PasswordHash: string(hash),
			Role:         u.role,
			Active:       true,
			CreatedAt:    time.Now(),
		}
		_ = s.CreateUser(user)
		userIDs = append(userIDs, user.ID)
	}

	adminID := userIDs[0]
	analystID := userIDs[1]

	type seedRecord struct {
		amount      float64
		rtype       models.RecordType
		category    string
		date        time.Time
		description string
		userID      string
	}

	now := time.Now()
	records := []seedRecord{
		{12500.00, models.RecordIncome, "Salary", now.AddDate(0, -2, -5), "Monthly salary - February", adminID},
		{3200.00, models.RecordExpense, "Rent", now.AddDate(0, -2, -3), "Office rent - February", adminID},
		{850.00, models.RecordExpense, "Utilities", now.AddDate(0, -2, -2), "Electricity and internet - Feb", adminID},
		{5000.00, models.RecordIncome, "Freelance", now.AddDate(0, -2, 0), "Web design project", analystID},
		{1200.00, models.RecordExpense, "Software", now.AddDate(0, -1, -20), "Annual SaaS subscriptions", analystID},
		{18000.00, models.RecordIncome, "Salary", now.AddDate(0, -1, -15), "Monthly salary - March", adminID},
		{3200.00, models.RecordExpense, "Rent", now.AddDate(0, -1, -10), "Office rent - March", adminID},
		{420.00, models.RecordExpense, "Travel", now.AddDate(0, -1, -8), "Client visit flights", analystID},
		{2500.00, models.RecordIncome, "Consulting", now.AddDate(0, -1, -5), "Strategy consulting fee", analystID},
		{650.00, models.RecordExpense, "Marketing", now.AddDate(0, 0, -20), "Social media ads campaign", adminID},
		{300.00, models.RecordExpense, "Office Supplies", now.AddDate(0, 0, -15), "Stationery and equipment", analystID},
		{7500.00, models.RecordIncome, "Investment", now.AddDate(0, 0, -10), "Dividend income Q1", adminID},
		{18000.00, models.RecordIncome, "Salary", now.AddDate(0, 0, -5), "Monthly salary - April", adminID},
		{3200.00, models.RecordExpense, "Rent", now.AddDate(0, 0, -4), "Office rent - April", adminID},
		{980.00, models.RecordExpense, "Utilities", now.AddDate(0, 0, -3), "Electricity and internet - Apr", adminID},
		{4200.00, models.RecordIncome, "Freelance", now.AddDate(0, 0, -2), "Mobile app UI project", analystID},
		{550.00, models.RecordExpense, "Travel", now.AddDate(0, 0, -1), "Conference attendance", analystID},
	}

	for _, r := range records {
		now2 := time.Now()
		_ = s.CreateRecord(&models.FinancialRecord{
			ID:          uuid.NewString(),
			UserID:      r.userID,
			Amount:      r.amount,
			Type:        r.rtype,
			Category:    r.category,
			Date:        r.date,
			Description: r.description,
			CreatedAt:   now2,
			UpdatedAt:   now2,
		})
	}
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	s := store.New()
	seedData(s)

	e := echo.New()
	e.HideBanner = true
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = customErrorHandler

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))
	e.Use(mw.RateLimit(50, 100)) // 50 req/s, burst of 100

	// Swagger UI
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Handlers
	authH := handlers.NewAuthHandler(s)
	userH := handlers.NewUserHandler(s)
	recordH := handlers.NewRecordHandler(s)
	dashH := handlers.NewDashboardHandler(s)

	api := e.Group("/api/v1")

	// ── Public ───────────────────────────────────────────────────────────────
	api.POST("/auth/login", authH.Login)

	// ── Protected (JWT required) ──────────────────────────────────────────────
	protected := api.Group("", mw.Auth(s))

	// Auth utilities
	protected.POST("/auth/logout", authH.Logout)
	protected.GET("/auth/me", authH.Me)

	// Users — admin only
	adminOnly := protected.Group("", mw.RequireRole(models.RoleAdmin))
	adminOnly.GET("/users", userH.ListUsers)
	adminOnly.GET("/users/:id", userH.GetUser)
	adminOnly.POST("/users", userH.CreateUser)
	adminOnly.PUT("/users/:id", userH.UpdateUser)
	adminOnly.DELETE("/users/:id", userH.DeleteUser)

	// Records
	// Read/Write: analyst + admin | Delete: admin only
	analystAdminRoles := mw.RequireRole(models.RoleAnalyst, models.RoleAdmin)
	writeRoles := mw.RequireRole(models.RoleAnalyst, models.RoleAdmin)
	adminRole := mw.RequireRole(models.RoleAdmin)

	protected.GET("/records", recordH.ListRecords, analystAdminRoles)
	protected.GET("/records/:id", recordH.GetRecord, analystAdminRoles)
	protected.POST("/records", recordH.CreateRecord, writeRoles)
	protected.PUT("/records/:id", recordH.UpdateRecord, writeRoles)
	protected.DELETE("/records/:id", recordH.DeleteRecord, adminRole)

	// Dashboard — all roles (viewer can only view dashboard data)
	dashRoles := mw.RequireRole(models.RoleViewer, models.RoleAnalyst, models.RoleAdmin)
	protected.GET("/dashboard/summary", dashH.GetSummary, dashRoles)
	protected.GET("/dashboard/weekly", dashH.GetWeeklyTrends, dashRoles)

	fmt.Println("🚀  Server running  →  http://localhost:8080")
	fmt.Println("📖  Swagger UI      →  http://localhost:8080/swagger/index.html")
	fmt.Println("")
	fmt.Println("Dummy credentials:")
	fmt.Println("  admin@zorvyn.io    / Admin@123    (role: admin)")
	fmt.Println("  analyst@zorvyn.io  / Analyst@123  (role: analyst)")
	fmt.Println("  viewer@zorvyn.io   / Viewer@123   (role: viewer)")

	e.Logger.Fatal(e.Start(":8080"))
}
