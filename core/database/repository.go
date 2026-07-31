package database

import (
"gorm.io/gorm"
)

// BaseRepository temel repository
type BaseRepository struct {
db *gorm.DB
}

// NewBaseRepository yeni repository oluşturur
func NewBaseRepository(db *gorm.DB) *BaseRepository {
return &BaseRepository{
db: db,
}
}

// GetDB veritabanı bağlantısını döndürür
func (r *BaseRepository) GetDB() *gorm.DB {
return r.db
}

// Create kayıt oluşturur
func (r *BaseRepository) Create(value interface{}) error {
return r.db.Create(value).Error
}

// Find kayıt bulur
func (r *BaseRepository) Find(dest interface{}, conditions ...interface{}) error {
return r.db.Find(dest, conditions...).Error
}

// First ilk kaydı bulur
func (r *BaseRepository) First(dest interface{}, conditions ...interface{}) error {
return r.db.First(dest, conditions...).Error
}

// Update kayıt günceller
func (r *BaseRepository) Update(model interface{}, updates interface{}) error {
return r.db.Model(model).Updates(updates).Error
}

// Delete kayıt siler
func (r *BaseRepository) Delete(value interface{}, conditions ...interface{}) error {
return r.db.Delete(value, conditions...).Error
}
