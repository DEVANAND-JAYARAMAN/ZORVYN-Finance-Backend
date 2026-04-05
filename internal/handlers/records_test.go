package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"zorvyn/internal/models"
	"zorvyn/internal/store"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type testValidator struct {
	v *validator.Validate
}

func (tv *testValidator) Validate(i interface{}) error {
	return tv.v.Struct(i)
}

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}
	return e
}

func errorCodeFromHTTPError(t *testing.T, err error) string {
	t.Helper()
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}

	switch msg := he.Message.(type) {
	case models.ErrorResponse:
		return msg.Code
	case map[string]interface{}:
		if v, ok := msg["code"].(string); ok {
			return v
		}
	}
	t.Fatalf("unexpected error message type: %T", he.Message)
	return ""
}

func TestListRecordsRejectsInvalidTypeQuery(t *testing.T) {
	e := newTestEcho()
	h := NewRecordHandler(store.New())

	req := httptest.NewRequest(http.MethodGet, "/records?type=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListRecords(c)
	if err == nil {
		t.Fatal("expected an error for invalid type")
	}
	if got := errorCodeFromHTTPError(t, err); got != "INVALID_QUERY" {
		t.Fatalf("expected INVALID_QUERY, got %s", got)
	}
}

func TestListRecordsRejectsInvalidDateRange(t *testing.T) {
	e := newTestEcho()
	h := NewRecordHandler(store.New())

	req := httptest.NewRequest(http.MethodGet, "/records?from=2026-04-10T00:00:00Z&to=2026-04-01T00:00:00Z", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListRecords(c)
	if err == nil {
		t.Fatal("expected an error for invalid date range")
	}
	if got := errorCodeFromHTTPError(t, err); got != "INVALID_DATE_RANGE" {
		t.Fatalf("expected INVALID_DATE_RANGE, got %s", got)
	}
}

func TestListRecordsRejectsNonIntegerPagination(t *testing.T) {
	e := newTestEcho()
	h := NewRecordHandler(store.New())

	req := httptest.NewRequest(http.MethodGet, "/records?page=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListRecords(c)
	if err == nil {
		t.Fatal("expected an error for non-integer page")
	}
	if got := errorCodeFromHTTPError(t, err); got != "INVALID_QUERY" {
		t.Fatalf("expected INVALID_QUERY, got %s", got)
	}
}

func TestUpdateRecordAllowsClearingDescription(t *testing.T) {
	e := newTestEcho()
	s := store.New()
	h := NewRecordHandler(s)

	now := time.Now()
	record := &models.FinancialRecord{
		ID:          "r1",
		UserID:      "u1",
		Amount:      200,
		Type:        models.RecordIncome,
		Category:    "Salary",
		Date:        now,
		Description: "to be cleared",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.CreateRecord(record); err != nil {
		t.Fatalf("failed to seed record: %v", err)
	}

	body := strings.NewReader(`{"description":""}`)
	req := httptest.NewRequest(http.MethodPut, "/records/r1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/records/:id")
	c.SetParamNames("id")
	c.SetParamValues("r1")

	if err := h.UpdateRecord(c); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var updated models.FinancialRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if updated.Description != "" {
		t.Fatalf("expected description to be cleared, got %q", updated.Description)
	}
}
