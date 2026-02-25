// Package repository_test ...
package repository_test

import (
	"context"
	"sync"
	"testing"
	"wallet/internal/repository"
	"wallet/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_ConcurrentDeposit(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	walletID := createWallet(t, pool, 0)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			err := store.RunInTx(ctx, func(ctx context.Context) error {
				balance, err := repo.GetBalanceForUpdate(ctx, walletID)
				if err != nil {
					return err
				}
				return repo.UpdateBalance(ctx, walletID, balance+1)
			})
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	balance, err := repo.GetBalance(ctx, walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(goroutines), balance)
}

func TestRepository_ConcurrentWithdraw(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	const goroutines = 50
	const initialBalance = int64(goroutines) // ровно столько чтобы хватило на всех

	walletID := createWallet(t, pool, initialBalance)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			_ = store.RunInTx(ctx, func(ctx context.Context) error {
				balance, err := repo.GetBalanceForUpdate(ctx, walletID)
				if err != nil {
					return err
				}
				if balance < 1 {
					return nil // недостаточно средств — выходим без ошибки
				}
				return repo.UpdateBalance(ctx, walletID, balance-1)
			})
		}()
	}

	wg.Wait()

	balance, err := repo.GetBalance(ctx, walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), balance,
		"ожидали баланс 0, получили %d", balance)
}

func TestRepository_ConcurrentMixed(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	const goroutines = 100
	const initialBalance = int64(goroutines / 2) // хватит на все списания

	walletID := createWallet(t, pool, initialBalance)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()

			err := store.RunInTx(ctx, func(ctx context.Context) error {
				balance, err := repo.GetBalanceForUpdate(ctx, walletID)
				if err != nil {
					return err
				}

				var newBalance int64
				if i%2 == 0 {
					// чётные — депозит
					newBalance = balance + 1
				} else {
					// нечётные — списание
					if balance < 1 {
						return nil
					}
					newBalance = balance - 1
				}

				if err := repo.UpdateBalance(ctx, walletID, newBalance); err != nil {
					return err
				}

				return repo.SaveOperation(ctx, usecase.Operation{
					ID:       uuid.New(),
					WalletID: walletID,
					Type:     map[bool]string{true: "DEPOSIT", false: "WITHDRAW"}[i%2 == 0],
					Amount:   1,
				})
			})
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	balance, err := repo.GetBalance(ctx, walletID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, balance, int64(0),
		"баланс ушёл в минус: %d", balance)
}
