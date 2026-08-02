package service

import (
"errors"
"fmt"

"modules/wallet/model"
"modules/wallet/repository"

"gorm.io/gorm"
)

type WalletService struct {
walletRepo *repository.WalletRepository
}

func NewWalletService(db *gorm.DB) *WalletService {
return &WalletService{
walletRepo: repository.NewWalletRepository(db),
}
}

// GetOrCreateWallet kullanÄ±cÄ±nÄ±n cÃ¼zdanÄ±nÄ± getirir veya oluÅŸturur
func (s *WalletService) GetOrCreateWallet(userID uint) (*model.WalletResponse, error) {
wallet, err := s.walletRepo.FindByUserID(userID)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
// Yeni cÃ¼zdan oluÅŸtur
wallet = &model.Wallet{
UserID:  userID,
Balance: 0,
}
if err := s.walletRepo.Create(wallet); err != nil {
return nil, err
}
return &model.WalletResponse{
Balance:     0,
TotalEarned: 0,
TotalSpent:  0,
}, nil
}
return nil, err
}
return &model.WalletResponse{
Balance:     wallet.Balance,
TotalEarned: wallet.TotalEarned,
TotalSpent:  wallet.TotalSpent,
}, nil
}

// EarnToken jeton kazandÄ±rÄ±r
func (s *WalletService) EarnToken(userID uint, req *model.EarnRequest) (*model.TransactionResponse, error) {
wallet, err := s.walletRepo.FindByUserID(userID)
if err != nil {
return nil, err
}

// Ä°ÅŸlem oluÅŸtur
tx := &model.WalletTransaction{
UserID:      userID,
Type:        req.Type,
Amount:      req.Amount,
Description: req.Description,
Balance:     wallet.Balance + req.Amount,
}

// CÃ¼zdanÄ± gÃ¼ncelle
wallet.Balance += req.Amount
wallet.TotalEarned += req.Amount

if err := s.walletRepo.Update(wallet); err != nil {
return nil, err
}

if err := s.walletRepo.CreateTransaction(tx); err != nil {
return nil, err
}

return s.toTransactionResponse(tx), nil
}

// SpendToken jeton harcar
func (s *WalletService) SpendToken(userID uint, req *model.SpendRequest) (*model.TransactionResponse, error) {
wallet, err := s.walletRepo.FindByUserID(userID)
if err != nil {
return nil, err
}

// Bakiye kontrolÃ¼
if wallet.Balance < req.Amount {
return nil, errors.New("insufficient balance")
}

// Ä°ÅŸlem oluÅŸtur
tx := &model.WalletTransaction{
UserID:      userID,
Type:        req.Type,
Amount:      -req.Amount,
Description: req.Description,
Balance:     wallet.Balance - req.Amount,
}

// CÃ¼zdanÄ± gÃ¼ncelle
wallet.Balance -= req.Amount
wallet.TotalSpent += req.Amount

if err := s.walletRepo.Update(wallet); err != nil {
return nil, err
}

if err := s.walletRepo.CreateTransaction(tx); err != nil {
return nil, err
}

return s.toTransactionResponse(tx), nil
}

// GetBalance kullanÄ±cÄ±nÄ±n bakiyesini getirir
func (s *WalletService) GetBalance(userID uint) (int, error) {
wallet, err := s.walletRepo.FindByUserID(userID)
if err != nil {
return 0, err
}
return wallet.Balance, nil
}

// GetTransactions kullanÄ±cÄ±nÄ±n iÅŸlem geÃ§miÅŸini getirir
func (s *WalletService) GetTransactions(userID uint, limit int) ([]model.TransactionResponse, error) {
transactions, err := s.walletRepo.GetTransactions(userID, limit)
if err != nil {
return nil, err
}

responses := make([]model.TransactionResponse, len(transactions))
for i, tx := range transactions {
responses[i] = *s.toTransactionResponse(&tx)
}
return responses, nil
}

// GetDailyBonus gÃ¼nlÃ¼k bonus kontrolÃ¼
func (s *WalletService) GetDailyBonus(userID uint) (bool, error) {
// TODO: Redis veya veritabanÄ±nda gÃ¼nlÃ¼k kontrol
return true, nil
}

// ClaimDailyBonus gÃ¼nlÃ¼k bonus al
func (s *WalletService) ClaimDailyBonus(userID uint) (*model.TransactionResponse, error) {
// GÃ¼nlÃ¼k bonus miktarÄ±
bonusAmount := 5

// Bonus zaten alÄ±nmÄ±ÅŸ mÄ± kontrol et
canClaim, err := s.GetDailyBonus(userID)
if err != nil {
return nil, err
}
if !canClaim {
return nil, errors.New("daily bonus already claimed")
}

req := &model.EarnRequest{
Amount:      bonusAmount,
Type:        "daily_bonus",
Description: fmt.Sprintf("GÃ¼nlÃ¼k giriÅŸ bonusu: %d jeton", bonusAmount),
}

return s.EarnToken(userID, req)
}

// toTransactionResponse model'den response oluÅŸturur
func (s *WalletService) toTransactionResponse(tx *model.WalletTransaction) *model.TransactionResponse {
return &model.TransactionResponse{
ID:          tx.ID,
Type:        tx.Type,
Amount:      tx.Amount,
Description: tx.Description,
Balance:     tx.Balance,
CreatedAt:   tx.CreatedAt,
}
}
