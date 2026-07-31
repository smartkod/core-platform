package service

import (
"errors"
"fmt"
"io"
"mime/multipart"
"os"
"path/filepath"
"strings"
"time"

"modules/filestorage/model"
"modules/filestorage/repository"

"gorm.io/gorm"
)

type FileService struct {
fileRepo   *repository.FileRepository
uploadPath string
baseURL    string
maxSize    int64
}

func NewFileService(db *gorm.DB, uploadPath, baseURL string) *FileService {
return &FileService{
fileRepo:   repository.NewFileRepository(db),
uploadPath: uploadPath,
baseURL:    baseURL,
maxSize:    10 * 1024 * 1024, // 10MB
}
}

// UploadFile dosya yükler
func (s *FileService) UploadFile(userID uint, fileHeader *multipart.FileHeader, req *model.UploadRequest) (*model.FileResponse, error) {
// Dosya boyutu kontrolü
if fileHeader.Size > s.maxSize {
return nil, errors.New("file size exceeds limit (10MB)")
}

// Dosya türü kontrolü
allowedTypes := map[string][]string{
"image":    {"image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml"},
"video":    {"video/mp4", "video/webm", "video/ogg"},
"document": {"application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
"audio":    {"audio/mpeg", "audio/ogg", "audio/wav"},
"other":    {},
}

mimeType := fileHeader.Header.Get("Content-Type")
if req.FileType != "other" {
allowed, ok := allowedTypes[req.FileType]
if ok && len(allowed) > 0 {
found := false
for _, mime := range allowed {
if mimeType == mime {
found = true
break
}
}
if !found {
return nil, fmt.Errorf("invalid file type for %s: %s", req.FileType, mimeType)
}
}
}

// Dosya adı oluştur
ext := filepath.Ext(fileHeader.Filename)
name := strings.TrimSuffix(fileHeader.Filename, ext)
timestamp := time.Now().UnixNano()
newFileName := fmt.Sprintf("%d_%d_%s%s", userID, timestamp, name, ext)

// Yolu oluştur
datePath := time.Now().Format("2006/01/02")
fullPath := filepath.Join(s.uploadPath, datePath)

// Klasörü oluştur
if err := os.MkdirAll(fullPath, 0755); err != nil {
return nil, err
}

// Dosyayı kaydet
filePath := filepath.Join(fullPath, newFileName)
src, err := fileHeader.Open()
if err != nil {
return nil, err
}
defer src.Close()

dst, err := os.Create(filePath)
if err != nil {
return nil, err
}
defer dst.Close()

if _, err := io.Copy(dst, src); err != nil {
return nil, err
}

// URL oluştur
url := fmt.Sprintf("%s/uploads/%s/%s", s.baseURL, datePath, newFileName)

// Veritabanına kaydet
file := &model.File{
UserID:      userID,
FileName:    fileHeader.Filename,
FileSize:    fileHeader.Size,
FileType:    req.FileType,
MimeType:    mimeType,
Path:        filePath,
URL:         url,
StorageType: req.StorageType,
IsPublic:    req.IsPublic,
}

if err := s.fileRepo.Create(file); err != nil {
os.Remove(filePath)
return nil, err
}

return s.toResponse(file), nil
}

// GetFile dosya bilgilerini getirir
func (s *FileService) GetFile(userID, fileID uint) (*model.FileResponse, error) {
file, err := s.fileRepo.FindByIDAndUser(fileID, userID)
if err != nil {
return nil, errors.New("file not found")
}
return s.toResponse(file), nil
}

// GetUserFiles kullanıcının dosyalarını getirir
func (s *FileService) GetUserFiles(userID uint, limit int) ([]model.FileResponse, error) {
files, err := s.fileRepo.FindByUserID(userID, limit)
if err != nil {
return nil, err
}

responses := make([]model.FileResponse, len(files))
for i, file := range files {
responses[i] = *s.toResponse(&file)
}
return responses, nil
}

// DeleteFile dosya siler
func (s *FileService) DeleteFile(userID, fileID uint) error {
file, err := s.fileRepo.FindByIDAndUser(fileID, userID)
if err != nil {
return errors.New("file not found")
}

// Dosyayı sil
if err := os.Remove(file.Path); err != nil {
// Dosya zaten silinmiş olabilir
}

// Veritabanından sil
return s.fileRepo.Delete(fileID)
}

// toResponse model'den response oluşturur
func (s *FileService) toResponse(file *model.File) *model.FileResponse {
return &model.FileResponse{
ID:          file.ID,
FileName:    file.FileName,
FileSize:    file.FileSize,
FileType:    file.FileType,
MimeType:    file.MimeType,
URL:         file.URL,
StorageType: file.StorageType,
IsPublic:    file.IsPublic,
CreatedAt:   file.CreatedAt,
}
}
