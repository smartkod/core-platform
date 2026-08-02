package api

import (
    "net/http"

    "github.com/labstack/echo/v4"
)

// Response standart API yanÄ±tÄ±
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Message string      `json:"message,omitempty"`
    Code    int         `json:"code"`
}

// SuccessResponse baÅŸarÄ±lÄ± yanÄ±t oluÅŸturur
func SuccessResponse(c echo.Context, data interface{}) error {
    return c.JSON(http.StatusOK, Response{
        Success: true,
        Data:    data,
        Code:    http.StatusOK,
    })
}

// CreatedResponse oluÅŸturma yanÄ±tÄ±
func CreatedResponse(c echo.Context, data interface{}) error {
    return c.JSON(http.StatusCreated, Response{
        Success: true,
        Data:    data,
        Code:    http.StatusCreated,
    })
}

// ErrorResponse hata yanÄ±tÄ± oluÅŸturur
func ErrorResponse(c echo.Context, statusCode int, message string) error {
    return c.JSON(statusCode, Response{
        Success: false,
        Error:   message,
        Code:    statusCode,
    })
}

// BadRequestResponse geÃ§ersiz istek yanÄ±tÄ±
func BadRequestResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusBadRequest, message)
}

// NotFoundResponse bulunamadÄ± yanÄ±tÄ±
func NotFoundResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusNotFound, message)
}

// UnauthorizedResponse yetkisiz yanÄ±tÄ±
func UnauthorizedResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusUnauthorized, message)
}

// ForbiddenResponse yasak yanÄ±tÄ±
func ForbiddenResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusForbidden, message)
}

// InternalServerErrorResponse sunucu hatasÄ± yanÄ±tÄ±
func InternalServerErrorResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusInternalServerError, message)
}
