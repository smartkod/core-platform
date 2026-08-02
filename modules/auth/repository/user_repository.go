package repository

import (
"core-platform/modules/auth/model"

"gorm.io/gorm"
)

// UserRepository kullanÃ„Â±cÃ„Â± repository'si
type UserRepository struct {
db *gorm.DB
}

// NewUserRepository yeni repository oluÃ…Å¸turur
func NewUserRepository(db *gorm.DB) *UserRepository {
return &UserRepository{
db: db,
}
}

// Create kullanÃ„Â±cÃ„Â± oluÃ…Å¸turur
func (r *UserRepository) Create(user *model.User) error {
return r.db.Create(user).Error
}

// FindByEmail email ile kullanÃ„Â±cÃ„Â± bulur
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
var user model.User
err := r.db.Where("email = ?", email).First(&user).Error
if err != nil {
return nil, err
}
return &user, nil
}

// FindByID ID ile kullanÃ„Â±cÃ„Â± bulur
func (r *UserRepository) FindByID(id uint) (*model.User, error) {
var user model.User
err := r.db.First(&user, id).Error
if err != nil {
return nil, err
}
return &user, nil
}

// Update kullanÃ„Â±cÃ„Â± gÃƒÂ¼nceller
func (r *UserRepository) Update(user *model.User) error {
return r.db.Save(user).Error
}

// Delete kullanÃ„Â±cÃ„Â± siler
func (r *UserRepository) Delete(id uint) error {
return r.db.Delete(&model.User{}, id).Error
}
