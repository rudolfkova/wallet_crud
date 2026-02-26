// Package usecase_test ...
package usecase_test

import (
	"context"
	"errors"
	"testing"
	walleterror "wallet/internal/error"
	"wallet/internal/mocks"
	"wallet/internal/model"
	"wallet/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testUUID() uuid.UUID {
	id, _ := uuid.Parse("11111111-1111-1111-1111-111111111111")
	return id
}

func setupTxManager(txm *mocks.TxManager) {
	txm.
		On("RunInTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		Return(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
}

// --- Balance ---

func TestUsecase_Balance_Success(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	expected := int64(1000)

	repo.On("GetBalance", ctx, walletID).Return(expected, nil)

	u := usecase.New(repo, txm)
	balance, err := u.Balance(ctx, walletID)

	require.NoError(t, err)
	assert.Equal(t, expected, balance)
	repo.AssertExpectations(t)
}

func TestUsecase_Balance_NotFound(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()

	repo.On("GetBalance", ctx, walletID).Return(int64(0), walleterror.ErrWalletNotFound)

	u := usecase.New(repo, txm)
	balance, err := u.Balance(ctx, walletID)

	require.ErrorIs(t, err, walleterror.ErrWalletNotFound)
	assert.Equal(t, int64(0), balance)
	repo.AssertExpectations(t)
}

func TestUsecase_Balance_InternalError(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	dbErr := errors.New("connection refused")

	repo.On("GetBalance", ctx, walletID).Return(int64(0), dbErr)

	u := usecase.New(repo, txm)
	balance, err := u.Balance(ctx, walletID)

	require.ErrorIs(t, err, dbErr)
	assert.Equal(t, int64(0), balance)
	repo.AssertExpectations(t)
}

// --- Deposit ---

func TestUsecase_Deposit_Success(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	amount := int64(500)
	currentBalance := int64(1000)

	setupTxManager(txm)
	repo.
		On("GetBalanceForUpdate", ctx, walletID).
		Return(currentBalance, nil)
	repo.
		On("UpdateBalance", ctx, walletID, currentBalance+amount).
		Return(nil)
	repo.
		On("SaveOperation", ctx, mock.MatchedBy(func(op usecase.Operation) bool {
			return op.WalletID == walletID &&
				op.Type == "DEPOSIT" &&
				op.Amount == amount &&
				op.ID != uuid.Nil
		})).
		Return(nil)

	u := usecase.New(repo, txm)
	err := u.Deposit(ctx, model.DepositInput{WalletID: walletID, Amount: amount})

	require.NoError(t, err)
	repo.AssertExpectations(t)
	txm.AssertExpectations(t)
}

func TestUsecase_Deposit_WalletNotFound(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	amount := int64(500)

	setupTxManager(txm)
	repo.
		On("GetBalanceForUpdate", ctx, walletID).
		Return(int64(0), walleterror.ErrWalletNotFound)

	u := usecase.New(repo, txm)
	err := u.Deposit(ctx, model.DepositInput{WalletID: walletID, Amount: amount})

	require.ErrorIs(t, err, walleterror.ErrWalletNotFound)
	repo.AssertNotCalled(t, "UpdateBalance")
	repo.AssertNotCalled(t, "SaveOperation")
	repo.AssertExpectations(t)
}

func TestUsecase_Deposit_UpdateBalanceError(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	amount := int64(500)
	currentBalance := int64(1000)
	dbErr := errors.New("update failed")

	setupTxManager(txm)
	repo.
		On("GetBalanceForUpdate", ctx, walletID).
		Return(currentBalance, nil)
	repo.
		On("UpdateBalance", ctx, walletID, currentBalance+amount).
		Return(dbErr)

	u := usecase.New(repo, txm)
	err := u.Deposit(ctx, model.DepositInput{WalletID: walletID, Amount: amount})

	require.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t, "SaveOperation")
	repo.AssertExpectations(t)
}

func TestUsecase_Deposit_SaveOperationError(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	amount := int64(500)
	currentBalance := int64(1000)
	dbErr := errors.New("insert failed")

	setupTxManager(txm)
	repo.
		On("GetBalanceForUpdate", ctx, walletID).
		Return(currentBalance, nil)
	repo.
		On("UpdateBalance", ctx, walletID, currentBalance+amount).
		Return(nil)
	repo.
		On("SaveOperation", ctx, mock.MatchedBy(func(op usecase.Operation) bool {
			return op.WalletID == walletID && op.Type == "DEPOSIT"
		})).
		Return(dbErr)

	u := usecase.New(repo, txm)
	err := u.Deposit(ctx, model.DepositInput{WalletID: walletID, Amount: amount})

	require.ErrorIs(t, err, dbErr)
	repo.AssertExpectations(t)
}

// --- Withdraw ---

func TestUsecase_Withdraw_Success(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	amount := int64(200)
	currentBalance := int64(1000)

	setupTxManager(txm)
	repo.
		On("GetBalanceForUpdate", ctx, walletID).
		Return(currentBalance, nil)
	repo.
		On("UpdateBalance", ctx, walletID, currentBalance-amount).
		Return(nil)
	repo.
		On("SaveOperation", ctx, mock.MatchedBy(func(op usecase.Operation) bool {
			return op.WalletID == walletID &&
				op.Type == "WITHDRAW" &&
				op.Amount == amount &&
				op.ID != uuid.Nil
		})).
		Return(nil)

	u := usecase.New(repo, txm)
	err := u.Withdraw(ctx, model.WithdrawInput{WalletID: walletID, Amount: amount})

	require.NoError(t, err)
	repo.AssertExpectations(t)
	txm.AssertExpectations(t)
}

func TestUsecase_Withdraw_InsufficientFunds(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	amount := int64(9999)
	currentBalance := int64(100)

	setupTxManager(txm)
	repo.
		On("GetBalanceForUpdate", ctx, walletID).
		Return(currentBalance, nil)

	u := usecase.New(repo, txm)
	err := u.Withdraw(ctx, model.WithdrawInput{WalletID: walletID, Amount: amount})

	require.ErrorIs(t, err, walleterror.ErrInsufficientFunds)
	repo.AssertNotCalled(t, "UpdateBalance")
	repo.AssertNotCalled(t, "SaveOperation")
	repo.AssertExpectations(t)
}

func TestUsecase_Withdraw_WalletNotFound(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	amount := int64(100)

	setupTxManager(txm)
	repo.
		On("GetBalanceForUpdate", ctx, walletID).
		Return(int64(0), walleterror.ErrWalletNotFound)

	u := usecase.New(repo, txm)
	err := u.Withdraw(ctx, model.WithdrawInput{WalletID: walletID, Amount: amount})

	require.ErrorIs(t, err, walleterror.ErrWalletNotFound)
	repo.AssertNotCalled(t, "UpdateBalance")
	repo.AssertNotCalled(t, "SaveOperation")
	repo.AssertExpectations(t)
}

func TestUsecase_Withdraw_UpdateBalanceError(t *testing.T) {
	repo := new(mocks.WalletRepository)
	txm := new(mocks.TxManager)
	ctx := context.Background()

	walletID := testUUID()
	amount := int64(200)
	currentBalance := int64(1000)
	dbErr := errors.New("update failed")

	setupTxManager(txm)
	repo.
		On("GetBalanceForUpdate", ctx, walletID).
		Return(currentBalance, nil)
	repo.
		On("UpdateBalance", ctx, walletID, currentBalance-amount).
		Return(dbErr)

	u := usecase.New(repo, txm)
	err := u.Withdraw(ctx, model.WithdrawInput{WalletID: walletID, Amount: amount})

	require.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t, "SaveOperation")
	repo.AssertExpectations(t)
}
