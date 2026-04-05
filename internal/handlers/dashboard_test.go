package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"zorvyn/internal/models"
	"zorvyn/internal/store"

	"github.com/labstack/echo/v4"
)

func TestGetWeeklyTrendsReturnsTwelveWeeks(t *testing.T) {
	e := echo.New()
	s := store.New()
	h := NewDashboardHandler(s)

	now := time.Now()
	if err := s.CreateRecord(&models.FinancialRecord{
		ID:        "r1",
		UserID:    "u1",
		Amount:    1000,
		Type:      models.RecordIncome,
		Category:  "Salary",
		Date:      now,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("failed to seed record: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard/weekly", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.GetWeeklyTrends(c); err != nil {
		t.Fatalf("unexpected weekly trends error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var trends []models.WeeklyTrend
	if err := json.Unmarshal(rec.Body.Bytes(), &trends); err != nil {
		t.Fatalf("failed to decode trends: %v", err)
	}
	if len(trends) != 12 {
		t.Fatalf("expected exactly 12 weeks, got %d", len(trends))
	}
}
