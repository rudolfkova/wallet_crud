package model

import "github.com/google/uuid"

// BalanceResponse ...
type BalanceResponse struct {
	WalletID uuid.UUID `json:"walletId"`
	Balance  int64     `json:"balance"`
}
