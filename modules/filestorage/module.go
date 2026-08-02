package filestorage

import (
"modules/filestorage/api"
"modules/filestorage/model"
"modules/filestorage/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type FileStorageModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.FileHandler
service *service.FileService
router  *echo.Echo
}

func NewFileStorageModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *FileStorageModule {
base := module.NewBaseModule("filestorage", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

// Upload path ve base URL
uploadPath := "./uploads"
baseURL := "http://localhost:8081"

return &FileStorageModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
service:    service.NewFileService(db, uploadPath, baseURL),
}
}

func (m *FileStorageModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.handler = api.NewFileHandler(m.service)

m.logger.Info("File Storage module initialized")
return nil
}

func (m *FileStorageModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

fileGroup := m.router.Group("/api/files")
fileGroup.POST("/upload", m.handler.UploadFile)
fileGroup.GET("", m.handler.GetFiles)
fileGroup.GET("/:id", m.handler.GetFile)
fileGroup.DELETE("/:id", m.handler.DeleteFile)

m.logger.Info("File Storage module started - routes registered")
return nil
}

func (m *FileStorageModule) Stop() error {
m.logger.Info("File Storage module stopped")
return m.BaseModule.Stop()
}

func (m *FileStorageModule) Models() []interface{} {
return []interface{}{
&model.File{},
}
}
