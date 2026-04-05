package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"
	"zorvyn/internal/models"
	"zorvyn/internal/store"

	"github.com/labstack/echo/v4"
)

type DashboardHandler struct {
	store *store.Store
}

func NewDashboardHandler(s *store.Store) *DashboardHandler {
	return &DashboardHandler{store: s}
}

// GetSummary godoc
// @Summary      Dashboard summary
// @Description  Returns aggregated financial data: total income/expenses, net balance, category breakdowns, recent 5 records, and monthly trends. Accessible by all authenticated roles.
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.DashboardSummary
// @Failure      401  {object}  models.ErrorResponse
// @Failure      403  {object}  models.ErrorResponse
// @Router       /dashboard/summary [get]
func (h *DashboardHandler) GetSummary(c echo.Context) error {
	records := h.store.AllActiveRecords()

	summary := models.DashboardSummary{
		ByCategoryIncome:  make(map[string]float64),
		ByCategoryExpense: make(map[string]float64),
		RecentRecords:     []models.FinancialRecord{},
		MonthlyTrends:     []models.MonthlyTrend{},
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Date.After(records[j].Date)
	})

	monthlyMap := make(map[string]*models.MonthlyTrend)
	for _, r := range records {
		switch r.Type {
		case models.RecordIncome:
			summary.TotalIncome += r.Amount
			summary.ByCategoryIncome[r.Category] += r.Amount
		case models.RecordExpense:
			summary.TotalExpenses += r.Amount
			summary.ByCategoryExpense[r.Category] += r.Amount
		}
		key := fmt.Sprintf("%d-%02d", r.Date.Year(), r.Date.Month())
		if _, ok := monthlyMap[key]; !ok {
			monthlyMap[key] = &models.MonthlyTrend{Month: key}
		}
		if r.Type == models.RecordIncome {
			monthlyMap[key].Income += r.Amount
		} else {
			monthlyMap[key].Expense += r.Amount
		}
	}

	summary.NetBalance = summary.TotalIncome - summary.TotalExpenses

	limit := 5
	if len(records) < limit {
		limit = len(records)
	}
	for _, r := range records[:limit] {
		summary.RecentRecords = append(summary.RecentRecords, *r)
	}

	for _, t := range monthlyMap {
		summary.MonthlyTrends = append(summary.MonthlyTrends, *t)
	}
	sort.Slice(summary.MonthlyTrends, func(i, j int) bool {
		return summary.MonthlyTrends[i].Month < summary.MonthlyTrends[j].Month
	})

	return c.JSON(http.StatusOK, summary)
}

// GetWeeklyTrends godoc
// @Summary      Weekly trends
// @Description  Returns income and expense totals grouped by ISO week (YYYY-WNN) for the last 12 weeks. Accessible by all authenticated roles.
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   models.WeeklyTrend
// @Failure      401  {object}  models.ErrorResponse
// @Failure      403  {object}  models.ErrorResponse
// @Router       /dashboard/weekly [get]
func (h *DashboardHandler) GetWeeklyTrends(c echo.Context) error {
	records := h.store.AllActiveRecords()
	now := time.Now()
	weeklyMap := make(map[string]*models.WeeklyTrend)
	for i := 11; i >= 0; i-- {
		d := now.AddDate(0, 0, -7*i)
		year, week := d.ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)
		weeklyMap[key] = &models.WeeklyTrend{Week: key}
	}

	for _, r := range records {
		year, week := r.Date.ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)
		trend, ok := weeklyMap[key]
		if !ok {
			continue
		}
		if r.Type == models.RecordIncome {
			trend.Income += r.Amount
		} else {
			trend.Expense += r.Amount
		}
	}

	trends := make([]models.WeeklyTrend, 0, len(weeklyMap))
	for _, t := range weeklyMap {
		trends = append(trends, *t)
	}
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Week < trends[j].Week
	})

	return c.JSON(http.StatusOK, trends)
}
