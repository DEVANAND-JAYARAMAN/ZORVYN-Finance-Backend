package handlers

import (
	"net/http"
	"strconv"
	"zorvyn/internal/models"

	"github.com/labstack/echo/v4"
)

func bindAndValidate(c echo.Context, v interface{}) error {
	if err := c.Bind(v); err != nil {
		return httpErr(http.StatusBadRequest, "BIND_ERROR", err.Error())
	}
	if err := c.Validate(v); err != nil {
		return httpErr(http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	return nil
}

func httpErr(status int, code, message string) *echo.HTTPError {
	return &echo.HTTPError{
		Code:    status,
		Message: models.ErrorResponse{Code: code, Message: message},
	}
}

func clampPage(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return page, limit
}

func parseIntQuery(c echo.Context, key string) (int, error) {
	v := c.QueryParam(key)
	if v == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, httpErr(http.StatusBadRequest, "INVALID_QUERY", key+" must be an integer")
	}
	return n, nil
}
