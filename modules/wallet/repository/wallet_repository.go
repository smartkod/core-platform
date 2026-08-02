package repository

import (
"modules/wallet/model"

"gorm.io/gorm"
)

type WalletRepository struct {
db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepository {
return &WalletRepository{
db: db,
}
}

func (r *WalletRepository) FindByUserID(userID uint) (*model.Wallet, error) {
var wallet model.Wallet
err := r.db.Where("user_id = ?", userID).First(&wallet).Error
if err != nil {
return nil, err
}
return &wallet, nil
}

func (r *WalletRepository) Create(wallet *model.Wallet) error {
return r.db.Create(wallet).Error
}

func (r *WalletRepository) Update(wallet *model.Wallet) error {
return r.db.Save(wallet).Error
}

func (r *WalletRepository) CreateTransaction(tx *model.WalletTransaction) error {
return r.db.Create(tx).Error
}

func (r *WalletRepository) GetTransactions(userID uint, limit int) ([]model.WalletTransaction, error) {
var transactions []model.WalletTransaction
err := r.db.Where("user_id = ?", userID).
Order("created_at DESC").
Limit(limit).
Find(&transactions).Error
return transactions, err
}
