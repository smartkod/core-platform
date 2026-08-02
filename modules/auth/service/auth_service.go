package service

import (
"errors"
"time"

"core-platform/modules/auth/model"
"core-platform/modules/auth/repository"

"github.com/golang-jwt/jwt/v5"
"golang.org/x/crypto/bcrypt"
"gorm.io/gorm"
)

// AuthService auth servisi
type AuthService struct {
userRepo   *repository.UserRepository
jwtSecret  []byte
tokenExpiry time.Duration
}

// NewAuthService yeni auth servisi oluÃ…Å¸turur
func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
return &AuthService{
userRepo:   repository.NewUserRepository(db),
jwtSecret:  []byte(jwtSecret),
tokenExpiry: time.Hour * 24, // 24 saat
}
}

// Register kullanÃ„Â±cÃ„Â± kaydeder
func (s *AuthService) Register(req *model.RegisterRequest) (*model.UserResponse, error) {
// Email kontrolÃƒÂ¼
existingUser, _ := s.userRepo.FindByEmail(req.Email)
if existingUser != nil {
return nil, errors.New("email already registered")
}

// Ã…Âifreyi hash'le
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
if err != nil {
return nil, err
}

// KullanÃ„Â±cÃ„Â± oluÃ…Å¸tur
user := &model.User{
Email:    req.Email,
Password: string(hashedPassword),
Name:     req.Name,
Language: "tr",
IsActive: true,
}

if err := s.userRepo.Create(user); err != nil {
return nil, err
}

return &model.UserResponse{
ID:       user.ID,
Email:    user.Email,
Name:     user.Name,
Language: user.Language,
Avatar:   user.Avatar,
IsActive: user.IsActive,
}, nil
}

// Login kullanÃ„Â±cÃ„Â± giriÃ…Å¸i yapar
func (s *AuthService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
// KullanÃ„Â±cÃ„Â±yÃ„Â± bul
user, err := s.userRepo.FindByEmail(req.Email)
if err != nil {
return nil, errors.New("invalid email or password")
}

// Ã…Âifreyi kontrol et
if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
return nil, errors.New("invalid email or password")
}

// Token oluÃ…Å¸tur
accessToken, err := s.generateAccessToken(user)
if err != nil {
return nil, err
}

refreshToken, err := s.generateRefreshToken(user)
if err != nil {
return nil, err
}

return &model.LoginResponse{
AccessToken:  accessToken,
RefreshToken: refreshToken,
TokenType:    "Bearer",
ExpiresIn:    int64(s.tokenExpiry.Seconds()),
}, nil
}

// generateAccessToken access token oluÃ…Å¸turur
func (s *AuthService) generateAccessToken(user *model.User) (string, error) {
claims := jwt.MapClaims{
"user_id": user.ID,
"email":   user.Email,
"exp":     time.Now().Add(s.tokenExpiry).Unix(),
"iat":     time.Now().Unix(),
}

token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
return token.SignedString(s.jwtSecret)
}

// generateRefreshToken refresh token oluÃ…Å¸turur
func (s *AuthService) generateRefreshToken(user *model.User) (string, error) {
claims := jwt.MapClaims{
"user_id": user.ID,
"email":   user.Email,
"exp":     time.Now().Add(s.tokenExpiry * 7).Unix(), // 7 gÃƒÂ¼n
"iat":     time.Now().Unix(),
"refresh": true,
}

token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
return token.SignedString(s.jwtSecret)
}

// ValidateToken token doÃ„Å¸rular
func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
return nil, errors.New("unexpected signing method")
}
return s.jwtSecret, nil
})
}
