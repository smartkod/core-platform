package service

import (
"errors"
"fmt"
"io"
"mime/multipart"
"os"
"path/filepath"
"regexp"
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

// validateFileName dosya adÃ„Â±nÃ„Â± gÃƒÂ¼venlik aÃƒÂ§Ã„Â±sÃ„Â±ndan doÃ„Å¸rular
func (s *FileService) validateFileName(filename string) error {
// 1. Path traversal kontrolÃƒÂ¼
if strings.Contains(filename, "..") {
return errors.New("invalid file path")
}

// 2. Sadece gÃƒÂ¼venli karakterler (alfanumerik, nokta, tire, alt ÃƒÂ§izgi)
safe := regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(filename, "")
if safe != filename {
return errors.New("file name contains invalid characters")
}

// 3. Ãƒâ€¡ift uzantÃ„Â± kontrolÃƒÂ¼
ext := filepath.Ext(filename)
name := strings.TrimSuffix(filename, ext)
if strings.Contains(name, ".") {
return errors.New("multiple extensions not allowed")
}

// 4. UzantÃ„Â± uzunluÃ„Å¸u kontrolÃƒÂ¼
if len(ext) > 10 {
return errors.New("file extension too long")
}

// 5. BoÃ…Å¸ dosya adÃ„Â± kontrolÃƒÂ¼
if len(filename) == 0 || len(filename) > 255 {
return errors.New("invalid file name length")
}

return nil
}

// validateFileType dosya tipini doÃ„Å¸rular
func (s *FileService) validateFileType(fileHeader *multipart.FileHeader) error {
// Ã„Â°zin verilen MIME tipleri
allowedMimes := map[string]bool{
"image/jpeg":      true,
"image/png":       true,
"image/gif":       true,
"image/webp":      true,
"image/svg+xml":   true,
"application/pdf": true,
"application/msword": true,
"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
"text/plain": true,
}

mimeType := fileHeader.Header.Get("Content-Type")
if !allowedMimes[mimeType] {
return fmt.Errorf("file type %s is not allowed", mimeType)
}

return nil
}

// UploadFile dosya yÃƒÂ¼kler (gÃƒÂ¼venli versiyon)
func (s *FileService) UploadFile(userID uint, fileHeader *multipart.FileHeader, req *model.UploadRequest) (*model.FileResponse, error) {
// 1. Dosya boyutu kontrolÃƒÂ¼
if fileHeader.Size > s.maxSize {
return nil, fmt.Errorf("file size exceeds limit: %d MB", s.maxSize/1024/1024)
}

// 2. Dosya adÃ„Â± doÃ„Å¸rulama
if err := s.validateFileName(fileHeader.Filename); err != nil {
return nil, err
}

// 3. Dosya tipi doÃ„Å¸rulama
if err := s.validateFileType(fileHeader); err != nil {
return nil, err
}

// 4. GerÃƒÂ§ek dosya tipini kontrol et (iÃƒÂ§erik bazlÃ„Â±)
src, err := fileHeader.Open()
if err != nil {
return nil, err
}
defer src.Close()

// Ã„Â°lk 512 byte'Ã„Â± oku
header := make([]byte, 512)
if _, err := src.Read(header); err != nil && err != io.EOF {
return nil, err
}

// Tekrar baÃ…Å¸a sar
src.Seek(0, 0)

// GerÃƒÂ§ek MIME tipini tespit et
detectedType := http.DetectContentType(header)
if !strings.HasPrefix(detectedType, "image/") && 
   !strings.HasPrefix(detectedType, "application/pdf") &&
   !strings.HasPrefix(detectedType, "text/plain") {
return nil, fmt.Errorf("invalid file content: %s", detectedType)
}

// 5. GÃƒÂ¼venli dosya adÃ„Â± oluÃ…Å¸tur
ext := filepath.Ext(fileHeader.Filename)
name := strings.TrimSuffix(fileHeader.Filename, ext)
safeName := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(name, "_")
timestamp := time.Now().UnixNano()
newFileName := fmt.Sprintf("%d_%d_%s%s", userID, timestamp, safeName, ext)

// 6. Yolu oluÃ…Å¸tur
datePath := time.Now().Format("2006/01/02")
fullPath := filepath.Join(s.uploadPath, datePath)

// 7. KlasÃƒÂ¶rÃƒÂ¼ oluÃ…Å¸tur
if err := os.MkdirAll(fullPath, 0755); err != nil {
return nil, err
}

// 8. DosyayÃ„Â± kaydet
filePath := filepath.Join(fullPath, newFileName)
dst, err := os.Create(filePath)
if err != nil {
return nil, err
}
defer dst.Close()

if _, err := io.Copy(dst, src); err != nil {
os.Remove(filePath)
return nil, err
}

// 9. URL oluÃ…Å¸tur
url := fmt.Sprintf("%s/uploads/%s/%s", s.baseURL, datePath, newFileName)

// 10. VeritabanÃ„Â±na kaydet
file := &model.File{
UserID:      userID,
FileName:    newFileName,
OriginalName: fileHeader.Filename,
FileSize:    fileHeader.Size,
FileType:    req.FileType,
MimeType:    fileHeader.Header.Get("Content-Type"),
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

func (s *FileService) toResponse(file *model.File) *model.FileResponse {
return &model.FileResponse{
ID:           file.ID,
FileName:     file.FileName,
OriginalName: file.OriginalName,
FileSize:     file.FileSize,
FileType:     file.FileType,
MimeType:     file.MimeType,
URL:          file.URL,
StorageType:  file.StorageType,
IsPublic:     file.IsPublic,
CreatedAt:    file.CreatedAt,
}
}