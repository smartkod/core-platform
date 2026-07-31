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

// NewAuthService yeni auth servisi oluşturur
func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
return &AuthService{
userRepo:   repository.NewUserRepository(db),
jwtSecret:  []byte(jwtSecret),
tokenExpiry: time.Hour * 24, // 24 saat
}
}

// Register kullanıcı kaydeder
func (s *AuthService) Register(req *model.RegisterRequest) (*model.UserResponse, error) {
// Email kontrolü
existingUser, _ := s.userRepo.FindByEmail(req.Email)
if existingUser != nil {
return nil, errors.New("email already registered")
}

// Şifreyi hash'le
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
if err != nil {
return nil, err
}

// Kullanıcı oluştur
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

// Login kullanıcı girişi yapar
func (s *AuthService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
// Kullanıcıyı bul
user, err := s.userRepo.FindByEmail(req.Email)
if err != nil {
return nil, errors.New("invalid email or password")
}

// Şifreyi kontrol et
if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
return nil, errors.New("invalid email or password")
}

// Token oluştur
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

// generateAccessToken access token oluşturur
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

// generateRefreshToken refresh token oluşturur
func (s *AuthService) generateRefreshToken(user *model.User) (string, error) {
claims := jwt.MapClaims{
"user_id": user.ID,
"email":   user.Email,
"exp":     time.Now().Add(s.tokenExpiry * 7).Unix(), // 7 gün
"iat":     time.Now().Unix(),
"refresh": true,
}

token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
return token.SignedString(s.jwtSecret)
}

// ValidateToken token doğrular
func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
return nil, errors.New("unexpected signing method")
}
return s.jwtSecret, nil
})
}
