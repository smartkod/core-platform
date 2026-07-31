package api

import (
"net/http"
"strconv"

"modules/filestorage/model"
"modules/filestorage/service"

"github.com/labstack/echo/v4"
)

type FileHandler struct {
fileService *service.FileService
}

func NewFileHandler(fileService *service.FileService) *FileHandler {
return &FileHandler{
fileService: fileService,
}
}

// UploadFile dosya yükler
func (h *FileHandler) UploadFile(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

file, err := c.FormFile("file")
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "File is required",
})
}

fileType := c.FormValue("file_type")
if fileType == "" {
fileType = "other"
}

isPublic := c.FormValue("is_public") == "true"
storageType := c.FormValue("storage_type")
if storageType == "" {
storageType = "local"
}

req := &model.UploadRequest{
FileName:    file.Filename,
FileType:    fileType,
IsPublic:    isPublic,
StorageType: storageType,
}

response, err := h.fileService.UploadFile(userID, file, req)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, response)
}

// GetFiles kullanıcının dosyalarını getirir
func (h *FileHandler) GetFiles(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

limit := 50
if limitStr := c.QueryParam("limit"); limitStr != "" {
if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
limit = val
}
}

files, err := h.fileService.GetUserFiles(userID, limit)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, files)
}

// GetFile dosya bilgilerini getirir
func (h *FileHandler) GetFile(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid file ID",
})
}

file, err := h.fileService.GetFile(userID, uint(id))
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, file)
}

// DeleteFile dosya siler
func (h *FileHandler) DeleteFile(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid file ID",
})
}

if err := h.fileService.DeleteFile(userID, uint(id)); err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, map[string]string{
"message": "File deleted successfully",
})
}
