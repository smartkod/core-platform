package database

import (
"gorm.io/gorm"
)

// BaseRepository temel repository
type BaseRepository struct {
db *gorm.DB
}

// NewBaseRepository yeni repository oluÃ…Å¸turur
func NewBaseRepository(db *gorm.DB) *BaseRepository {
return &BaseRepository{
db: db,
}
}

// GetDB veritabanÃ„Â± baÃ„Å¸lantÃ„Â±sÃ„Â±nÃ„Â± dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (r *BaseRepository) GetDB() *gorm.DB {
return r.db
}

// Create kayÃ„Â±t oluÃ…Å¸turur
func (r *BaseRepository) Create(value interface{}) error {
return r.db.Create(value).Error
}

// Find kayÃ„Â±t bulur
func (r *BaseRepository) Find(dest interface{}, conditions ...interface{}) error {
return r.db.Find(dest, conditions...).Error
}

// First ilk kaydÃ„Â± bulur
func (r *BaseRepository) First(dest interface{}, conditions ...interface{}) error {
return r.db.First(dest, conditions...).Error
}

// Update kayÃ„Â±t gÃƒÂ¼nceller
func (r *BaseRepository) Update(model interface{}, updates interface{}) error {
return r.db.Model(model).Updates(updates).Error
}

// Delete kayÃ„Â±t siler
func (r *BaseRepository) Delete(value interface{}, conditions ...interface{}) error {
return r.db.Delete(value, conditions...).Error
}
