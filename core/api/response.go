package api

import (
    "net/http"

    "github.com/labstack/echo/v4"
)

// Response standart API yanıtı
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Message string      `json:"message,omitempty"`
    Code    int         `json:"code"`
}

// SuccessResponse başarılı yanıt oluşturur
func SuccessResponse(c echo.Context, data interface{}) error {
    return c.JSON(http.StatusOK, Response{
        Success: true,
        Data:    data,
        Code:    http.StatusOK,
    })
}

// CreatedResponse oluşturma yanıtı
func CreatedResponse(c echo.Context, data interface{}) error {
    return c.JSON(http.StatusCreated, Response{
        Success: true,
        Data:    data,
        Code:    http.StatusCreated,
    })
}

// ErrorResponse hata yanıtı oluşturur
func ErrorResponse(c echo.Context, statusCode int, message string) error {
    return c.JSON(statusCode, Response{
        Success: false,
        Error:   message,
        Code:    statusCode,
    })
}

// BadRequestResponse geçersiz istek yanıtı
func BadRequestResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusBadRequest, message)
}

// NotFoundResponse bulunamadı yanıtı
func NotFoundResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusNotFound, message)
}

// UnauthorizedResponse yetkisiz yanıtı
func UnauthorizedResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusUnauthorized, message)
}

// ForbiddenResponse yasak yanıtı
func ForbiddenResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusForbidden, message)
}

// InternalServerErrorResponse sunucu hatası yanıtı
func InternalServerErrorResponse(c echo.Context, message string) error {
    return ErrorResponse(c, http.StatusInternalServerError, message)
}
