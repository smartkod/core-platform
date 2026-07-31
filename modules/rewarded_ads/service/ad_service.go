package service

import (
"errors"
"fmt"
"time"

"modules/rewarded_ads/model"
"modules/rewarded_ads/repository"

"gorm.io/gorm"
)

// WalletService interface for wallet integration
type WalletService interface {
AddTokens(userID uint, amount int, reason, description string) error
}

type AdService struct {
adRepo    *repository.AdRepository
walletSvc WalletService
db        *gorm.DB
}

func NewAdService(db *gorm.DB, walletSvc WalletService) *AdService {
return &AdService{
adRepo:    repository.NewAdRepository(db),
walletSvc: walletSvc,
db:        db,
}
}

// GetAdStatus kullanıcının reklam durumunu getirir
func (s *AdService) GetAdStatus(userID uint) (*model.AdStatusResponse, error) {
today := time.Now().Format("2006-01-02")

limit, err := s.adRepo.GetDailyLimit(userID, today)
if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
return nil, err
}

watchedToday := 0
if limit != nil {
watchedToday = limit.TotalWatched
}

config := s.getAdConfig()

remaining := config.DailyLimit - watchedToday
if remaining < 0 {
remaining = 0
}

return &model.AdStatusResponse{
CanWatch:        remaining > 0,
WatchedToday:    watchedToday,
RemainingToday:  remaining,
TokenPerAd:      config.TokenPerAd,
DailyLimit:      config.DailyLimit,
DailyTokenLimit: config.DailyTokenLimit,
}, nil
}

// WatchAd reklam izleme işlemi
func (s *AdService) WatchAd(userID uint, req *model.AdWatchRequest) (*model.AdRewardResponse, error) {
today := time.Now().Format("2006-01-02")

// Günlük limit kontrolü
limit, err := s.adRepo.GetDailyLimit(userID, today)
if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
return nil, err
}

config := s.getAdConfig()

watchedToday := 0
if limit != nil {
watchedToday = limit.TotalWatched
}

if watchedToday >= config.DailyLimit {
return nil, errors.New("daily ad limit reached")
}

// Aynı reklam kontrolü
exists, err := s.adRepo.AdExists(userID, req.AdID, req.AdProvider)
if err != nil {
return nil, err
}
if exists {
return nil, errors.New("this ad has already been rewarded")
}

// Ödül hesapla
rewardValue := config.TokenPerAd

// Transaction başlat
tx := s.db.Begin()

// Ödül kaydı oluştur
reward := &model.AdReward{
UserID:      userID,
AdProvider:  req.AdProvider,
AdID:        req.AdID,
RewardType:  req.RewardType,
RewardValue: rewardValue,
IsVerified:  true,
VerifiedAt:  &[]time.Time{time.Now()}[0],
}

if err := s.adRepo.CreateReward(tx, reward); err != nil {
tx.Rollback()
return nil, err
}

// Günlük limit güncelle
if limit == nil {
limit = &model.DailyAdLimit{
UserID: userID,
Date:   today,
}
}
limit.TotalWatched++
limit.TotalReward += rewardValue

if err := s.adRepo.UpdateDailyLimit(tx, limit); err != nil {
tx.Rollback()
return nil, err
}

// Wallet'a token ekle
if err := s.walletSvc.AddTokens(userID, rewardValue, "ad_reward", fmt.Sprintf("Reklam ödülü: %s", req.AdID)); err != nil {
tx.Rollback()
return nil, err
}

if err := tx.Commit().Error; err != nil {
return nil, err
}

remaining := config.DailyLimit - limit.TotalWatched
if remaining < 0 {
remaining = 0
}

return &model.AdRewardResponse{
Success:     true,
Message:     fmt.Sprintf("Ad watched successfully! You earned %d tokens.", rewardValue),
RewardValue: rewardValue,
TotalToday:  limit.TotalWatched,
Remaining:   remaining,
}, nil
}

// getAdConfig reklam konfigürasyonunu getirir
func (s *AdService) getAdConfig() *model.AdConfig {
return &model.AdConfig{
DailyLimit:      5,
DailyTokenLimit: 50,
TokenPerAd:      10,
IsActive:        true,
}
}
