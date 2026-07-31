package repository

import (
"modules/filestorage/model"

"gorm.io/gorm"
)

type FileRepository struct {
db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
return &FileRepository{
db: db,
}
}

func (r *FileRepository) Create(file *model.File) error {
return r.db.Create(file).Error
}

func (r *FileRepository) FindByIDAndUser(fileID, userID uint) (*model.File, error) {
var file model.File
err := r.db.Where("id = ? AND user_id = ? AND is_deleted = ?", fileID, userID, false).First(&file).Error
if err != nil {
return nil, err
}
return &file, nil
}

func (r *FileRepository) FindByUserID(userID uint, limit int) ([]model.File, error) {
var files []model.File
err := r.db.Where("user_id = ? AND is_deleted = ?", userID, false).
Order("created_at DESC").
Limit(limit).
Find(&files).Error
return files, err
}

func (r *FileRepository) Delete(fileID uint) error {
return r.db.Model(&model.File{}).Where("id = ?", fileID).
Updates(map[string]interface{}{
"is_deleted": true,
"deleted_at": gorm.Expr("NOW()"),
}).Error
}
