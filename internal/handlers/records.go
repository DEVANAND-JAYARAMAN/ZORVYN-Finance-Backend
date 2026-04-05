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
)

type RecordHandler struct {
	store *store.Store
}

func NewRecordHandler(s *store.Store) *RecordHandler {
	return &RecordHandler{store: s}
}

// ListRecords godoc
// @Summary      List financial records
// @Description  Returns paginated financial records. Supports filtering by type, category, date range, and free-text search.
// @Description  Accessible by analyst and admin roles.
// @Tags         records
// @Produce      json
// @Security     BearerAuth
// @Param        type      query     string  false  "Filter by type"          Enums(income, expense)
// @Param        category  query     string  false  "Filter by category (case-insensitive)"
// @Param        search    query     string  false  "Search in category and description"
// @Param        from      query     string  false  "Start date filter (RFC3339, e.g. 2026-03-01T00:00:00Z)"
// @Param        to        query     string  false  "End date filter (RFC3339, e.g. 2026-04-30T23:59:59Z)"
// @Param        page      query     int     false  "Page number (default: 1)"
// @Param        limit     query     int     false  "Page size, max 100 (default: 20)"
// @Success      200       {object}  models.PaginatedRecords
// @Failure      400       {object}  models.ErrorResponse
// @Failure      401       {object}  models.ErrorResponse
// @Router       /records [get]
func (h *RecordHandler) ListRecords(c echo.Context) error {
	f := models.RecordFilter{
		Type:     models.RecordType(c.QueryParam("type")),
		Category: c.QueryParam("category"),
		Search:   c.QueryParam("search"),
	}
	if f.Type != "" && f.Type != models.RecordIncome && f.Type != models.RecordExpense {
		return httpErr(http.StatusBadRequest, "INVALID_QUERY", "type must be one of: income, expense")
	}
	if v := c.QueryParam("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return httpErr(http.StatusBadRequest, "INVALID_DATE", "invalid 'from' date, use RFC3339 format e.g. 2026-03-01T00:00:00Z")
		}
		f.From = &t
	}
	if v := c.QueryParam("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return httpErr(http.StatusBadRequest, "INVALID_DATE", "invalid 'to' date, use RFC3339 format e.g. 2026-04-30T23:59:59Z")
		}
		f.To = &t
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return httpErr(http.StatusBadRequest, "INVALID_DATE_RANGE", "'from' must be less than or equal to 'to'")
	}

	page, err := parseIntQuery(c, "page")
	if err != nil {
		return err
	}
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return err
	}
	f.Page = page
	f.Limit = limit
	f.Page, f.Limit = clampPage(f.Page, f.Limit)

	records, total := h.store.ListRecords(f)

	pages := total / f.Limit
	if total%f.Limit != 0 {
		pages++
	}

	return c.JSON(http.StatusOK, models.PaginatedRecords{
		Data:  records,
		Total: total,
		Page:  f.Page,
		Limit: f.Limit,
		Pages: pages,
	})
}

// GetRecord godoc
// @Summary      Get record by ID
// @Description  Returns a single financial record by its ID. Accessible by analyst and admin roles.
// @Tags         records
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Record UUID"
// @Success      200  {object}  models.FinancialRecord
// @Failure      404  {object}  models.ErrorResponse
// @Router       /records/{id} [get]
func (h *RecordHandler) GetRecord(c echo.Context) error {
	r, err := h.store.GetRecordByID(c.Param("id"))
	if err != nil {
		return httpErr(http.StatusNotFound, "NOT_FOUND", "record not found")
	}
	return c.JSON(http.StatusOK, r)
}

// CreateRecord godoc
// @Summary      Create financial record
// @Description  Creates a new financial record. Accessible by analyst and admin roles.
// @Tags         records
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      models.CreateRecordRequest  true  "Record payload"
// @Success      201   {object}  models.FinancialRecord
// @Failure      400   {object}  models.ErrorResponse
// @Failure      403   {object}  models.ErrorResponse
// @Router       /records [post]
func (h *RecordHandler) CreateRecord(c echo.Context) error {
	var req models.CreateRecordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	req.Category = strings.TrimSpace(req.Category)
	req.Description = strings.TrimSpace(req.Description)
	if req.Category == "" {
		return httpErr(http.StatusBadRequest, "VALIDATION_ERROR", "category is required")
	}
	caller := middleware.CurrentUser(c)
	now := time.Now()
	r := &models.FinancialRecord{
		ID:          uuid.NewString(),
		UserID:      caller.ID,
		Amount:      req.Amount,
		Type:        req.Type,
		Category:    req.Category,
		Date:        req.Date,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = h.store.CreateRecord(r)
	return c.JSON(http.StatusCreated, r)
}

// UpdateRecord godoc
// @Summary      Update financial record
// @Description  Partially updates a financial record. Only provided fields are changed. Accessible by analyst and admin.
// @Tags         records
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                      true  "Record UUID"
// @Param        body  body      models.UpdateRecordRequest  true  "Fields to update (all optional)"
// @Success      200   {object}  models.FinancialRecord
// @Failure      400   {object}  models.ErrorResponse
// @Failure      403   {object}  models.ErrorResponse
// @Failure      404   {object}  models.ErrorResponse
// @Router       /records/{id} [put]
func (h *RecordHandler) UpdateRecord(c echo.Context) error {
	r, err := h.store.GetRecordByID(c.Param("id"))
	if err != nil {
		return httpErr(http.StatusNotFound, "NOT_FOUND", "record not found")
	}
	var req models.UpdateRecordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	if req.Amount != nil {
		r.Amount = *req.Amount
	}
	if req.Type != nil {
		r.Type = *req.Type
	}
	if req.Category != nil {
		category := strings.TrimSpace(*req.Category)
		if category == "" {
			return httpErr(http.StatusBadRequest, "VALIDATION_ERROR", "category cannot be empty")
		}
		r.Category = category
	}
	if req.Date != nil {
		r.Date = *req.Date
	}
	if req.Description != nil {
		r.Description = strings.TrimSpace(*req.Description)
	}
	r.UpdatedAt = time.Now()
	if err := h.store.UpdateRecord(r); err != nil {
		return httpErr(http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, r)
}

// DeleteRecord godoc
// @Summary      Delete financial record
// @Description  Soft-deletes a financial record (it is hidden from all queries but not permanently removed). Admin only.
// @Tags         records
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Record UUID"
// @Success      204  "No Content"
// @Failure      403  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /records/{id} [delete]
func (h *RecordHandler) DeleteRecord(c echo.Context) error {
	if err := h.store.SoftDeleteRecord(c.Param("id")); err != nil {
		return httpErr(http.StatusNotFound, "NOT_FOUND", "record not found")
	}
	return c.NoContent(http.StatusNoContent)
}
