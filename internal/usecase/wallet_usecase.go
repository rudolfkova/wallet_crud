// Package usecase ...
package usecase

import (
	"context"
	walleterror "wallet/internal/error"
	"wallet/internal/model"

	"github.com/google/uuid"
)

// TxManager ...
type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// WalletRepository ...
type WalletRepository interface {
	// GetBalance ...
	GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error)
	// GetBalanceForUpdate ...
	GetBalanceForUpdate(ctx context.Context, walletID uuid.UUID) (int64, error)
	// UpdateBalance ...
	UpdateBalance(ctx context.Context, walletID uuid.UUID, newBalance int64) error
	// SaveOperation ...
	SaveOperation(ctx context.Context, op Operation) error
}

// WalletUsecase ...
type WalletUsecase struct {
	repo WalletRepository
	txm  TxManager
}

// New ...
func New(repo WalletRepository, txm TxManager) *WalletUsecase {
	return &WalletUsecase{
		repo: repo,
		txm:  txm,
	}
}

// Operation ...
type Operation struct {
	ID       uuid.UUID
	WalletID uuid.UUID
	Type     string // "DEPOSIT" или "WITHDRAW"
	Amount   int64
}

// Balance ...
func (u *WalletUsecase) Balance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	balance, err := u.repo.GetBalance(ctx, walletID)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

// Deposit ...
func (u *WalletUsecase) Deposit(ctx context.Context, in model.DepositInput) error {
	return u.txm.RunInTx(ctx, func(ctx context.Context) error {
		balance, err := u.repo.GetBalanceForUpdate(ctx, in.WalletID)
		if err != nil {
			return err
		}

		newBalance := balance + in.Amount

		if err := u.repo.UpdateBalance(ctx, in.WalletID, newBalance); err != nil {
			return err
		}

		op := Operation{
			ID:       uuid.New(),
			WalletID: in.WalletID,
			Type:     "DEPOSIT",
			Amount:   in.Amount,
		}
		return u.repo.SaveOperation(ctx, op)
	})
}

// Withdraw ...
func (u *WalletUsecase) Withdraw(ctx context.Context, in model.WithdrawInput) error {
	return u.txm.RunInTx(ctx, func(ctx context.Context) error {
		balance, err := u.repo.GetBalanceForUpdate(ctx, in.WalletID)
		if err != nil {
			return err
		}

		if balance < in.Amount {
			return walleterror.ErrInsufficientFunds
		}

		newBalance := balance - in.Amount

		if err := u.repo.UpdateBalance(ctx, in.WalletID, newBalance); err != nil {
			return err
		}

		op := Operation{
			ID:       uuid.New(),
			WalletID: in.WalletID,
			Type:     "WITHDRAW",
			Amount:   in.Amount,
		}
		return u.repo.SaveOperation(ctx, op)
	})
}
