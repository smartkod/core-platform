package repository

import (
"modules/users/model"

"gorm.io/gorm"
)

// UserRepository kullanıcı repository'si
type UserRepository struct {
db *gorm.DB
}

// NewUserRepository yeni repository oluşturur
func NewUserRepository(db *gorm.DB) *UserRepository {
return &UserRepository{
db: db,
}
}

// Create kullanıcı oluşturur
func (r *UserRepository) Create(user *model.User) error {
return r.db.Create(user).Error
}

// FindByEmail email ile kullanıcı bulur
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
var user model.User
err := r.db.Where("email = ?", email).First(&user).Error
if err != nil {
return nil, err
}
return &user, nil
}

// FindByID ID ile kullanıcı bulur
func (r *UserRepository) FindByID(id uint) (*model.User, error) {
var user model.User
err := r.db.First(&user, id).Error
if err != nil {
return nil, err
}
return &user, nil
}

// FindAll tüm kullanıcıları bulur
func (r *UserRepository) FindAll() ([]model.User, error) {
var users []model.User
err := r.db.Find(&users).Error
return users, err
}

// Update kullanıcı günceller
func (r *UserRepository) Update(user *model.User) error {
return r.db.Save(user).Error
}

// Delete kullanıcı siler
func (r *UserRepository) Delete(id uint) error {
return r.db.Delete(&model.User{}, id).Error
}
