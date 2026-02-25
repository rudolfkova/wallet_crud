package model

import "github.com/google/uuid"

// BalanceResponse ...
type BalanceResponse struct {
	WalletID uuid.UUID `json:"walletId"`
	Balance  int64     `json:"balance"`
}

// DepositInput ...
type DepositInput struct {
	WalletID uuid.UUID
	Amount   int64
}

// WithdrawInput ...
type WithdrawInput struct {
	WalletID uuid.UUID
	Amount   int64
}
