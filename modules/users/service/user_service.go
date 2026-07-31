package service

import (
"errors"

"modules/users/model"
"modules/users/repository"

"gorm.io/gorm"
)

// UserService kullanıcı servisi
type UserService struct {
userRepo *repository.UserRepository
}

// NewUserService yeni servis oluşturur
func NewUserService(db *gorm.DB) *UserService {
return &UserService{
userRepo: repository.NewUserRepository(db),
}
}

// GetUserByID ID ile kullanıcı getirir
func (s *UserService) GetUserByID(id uint) (*model.UserResponse, error) {
user, err := s.userRepo.FindByID(id)
if err != nil {
return nil, errors.New("user not found")
}
return s.toResponse(user), nil
}

// GetUserByEmail email ile kullanıcı getirir
func (s *UserService) GetUserByEmail(email string) (*model.UserResponse, error) {
user, err := s.userRepo.FindByEmail(email)
if err != nil {
return nil, errors.New("user not found")
}
return s.toResponse(user), nil
}

// GetAllUsers tüm kullanıcıları getirir
func (s *UserService) GetAllUsers() ([]model.UserResponse, error) {
users, err := s.userRepo.FindAll()
if err != nil {
return nil, err
}

responses := make([]model.UserResponse, len(users))
for i, user := range users {
responses[i] = *s.toResponse(&user)
}
return responses, nil
}

// UpdateUser kullanıcı günceller
func (s *UserService) UpdateUser(id uint, req *model.UpdateUserRequest) (*model.UserResponse, error) {
user, err := s.userRepo.FindByID(id)
if err != nil {
return nil, errors.New("user not found")
}

if req.Name != "" {
user.Name = req.Name
}
if req.Language != "" {
user.Language = req.Language
}
if req.Avatar != "" {
user.Avatar = req.Avatar
}

if err := s.userRepo.Update(user); err != nil {
return nil, err
}

return s.toResponse(user), nil
}

// DeleteUser kullanıcı siler
func (s *UserService) DeleteUser(id uint) error {
return s.userRepo.Delete(id)
}

// toResponse model'den response oluşturur
func (s *UserService) toResponse(user *model.User) *model.UserResponse {
return &model.UserResponse{
ID:        user.ID,
Email:     user.Email,
Name:      user.Name,
Language:  user.Language,
Avatar:    user.Avatar,
IsActive:  user.IsActive,
LastLogin: user.LastLogin,
CreatedAt: user.CreatedAt,
UpdatedAt: user.UpdatedAt,
}
}
