// Package repository_test ...
package repository_test

import (
	"context"
	"os"
	"testing"
	"wallet/internal/driver/sqlstore"
	walleterror "wallet/internal/error"
	"wallet/internal/repository"
	"wallet/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDB(t *testing.T) (*pgxpool.Pool, *sqlstore.Store) {
	t.Helper()

	err := godotenv.Load("../../config.env")
	if err != nil {
		t.Skip("env not load")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration tests")
	}

	ctx := context.Background()
	store, err := sqlstore.New(ctx, dsn, sqlstore.DefaultPoolConfig())
	require.NoError(t, err, "failed to connect to test database")

	t.Cleanup(func() { store.Close() })

	return store.Pool(), store
}

func createWallet(t *testing.T, pool *pgxpool.Pool, balance int64) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO wallets (id, balance) VALUES ($1, $2)`,
		id, balance,
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM wallets WHERE id = $1`, id)
	})

	return id
}

// --- GetBalance ---

func TestRepository_GetBalance_Success(t *testing.T) {
	pool, _ := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	walletID := createWallet(t, pool, 1000)

	balance, err := repo.GetBalance(ctx, walletID)

	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance)
}

func TestRepository_GetBalance_NotFound(t *testing.T) {
	pool, _ := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	balance, err := repo.GetBalance(ctx, uuid.New())

	assert.ErrorIs(t, err, walleterror.ErrWalletNotFound)
	assert.Equal(t, int64(0), balance)
}

// --- GetBalanceForUpdate ---

func TestRepository_GetBalanceForUpdate_InsideTx(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	walletID := createWallet(t, pool, 500)

	err := store.RunInTx(ctx, func(ctx context.Context) error {
		balance, err := repo.GetBalanceForUpdate(ctx, walletID)
		require.NoError(t, err)
		assert.Equal(t, int64(500), balance)
		return nil
	})

	require.NoError(t, err)
}

func TestRepository_GetBalanceForUpdate_NotFound(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	err := store.RunInTx(ctx, func(ctx context.Context) error {
		_, err := repo.GetBalanceForUpdate(ctx, uuid.New())
		return err
	})

	assert.ErrorIs(t, err, walleterror.ErrWalletNotFound)
}

// --- UpdateBalance ---

func TestRepository_UpdateBalance_Success(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	walletID := createWallet(t, pool, 0)

	err := store.RunInTx(ctx, func(ctx context.Context) error {
		return repo.UpdateBalance(ctx, walletID, 750)
	})
	require.NoError(t, err)

	balance, err := repo.GetBalance(ctx, walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(750), balance)
}

func TestRepository_UpdateBalance_CheckConstraint(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	walletID := createWallet(t, pool, 100)

	err := store.RunInTx(ctx, func(ctx context.Context) error {
		return repo.UpdateBalance(ctx, walletID, -1)
	})

	assert.Error(t, err)
}

// --- SaveOperation ---

func TestRepository_SaveOperation_Success(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	walletID := createWallet(t, pool, 0)
	opID := uuid.New()

	err := store.RunInTx(ctx, func(ctx context.Context) error {
		return repo.SaveOperation(ctx, usecase.Operation{
			ID:       opID,
			WalletID: walletID,
			Type:     "DEPOSIT",
			Amount:   300,
		})
	})
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_operations WHERE id = $1 AND wallet_id = $2`,
		opID, walletID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRepository_SaveOperation_InvalidType(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	walletID := createWallet(t, pool, 0)

	err := store.RunInTx(ctx, func(ctx context.Context) error {
		return repo.SaveOperation(ctx, usecase.Operation{
			ID:       uuid.New(),
			WalletID: walletID,
			Type:     "INVALID",
			Amount:   100,
		})
	})

	assert.Error(t, err)
}

// --- Транзакционность ---

func TestRepository_Rollback_OnError(t *testing.T) {
	pool, store := setupDB(t)
	repo := repository.New(pool)
	ctx := context.Background()

	walletID := createWallet(t, pool, 100)

	_ = store.RunInTx(ctx, func(ctx context.Context) error {
		err := repo.UpdateBalance(ctx, walletID, 9999)
		require.NoError(t, err)
		return assert.AnError
	})

	balance, err := repo.GetBalance(ctx, walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balance)
}
