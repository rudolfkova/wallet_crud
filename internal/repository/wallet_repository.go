// Package repository ...
package repository

import (
	"context"
	"errors"
	"fmt"
	txctx "wallet/internal/driver"
	walleterror "wallet/internal/error"
	"wallet/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// WalletRepository ...
type WalletRepository struct {
	pool *pgxpool.Pool
}

// New ...
func New(pool *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{pool: pool}
}

func (r *WalletRepository) q(ctx context.Context) querier {
	if tx, ok := txctx.ExtractTx(ctx); ok {
		return tx
	}
	return r.pool
}

// GetBalance ...
func (r *WalletRepository) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	return r.scanBalance(ctx, walletID, false)
}

// GetBalanceForUpdate ...
func (r *WalletRepository) GetBalanceForUpdate(ctx context.Context, walletID uuid.UUID) (int64, error) {
	return r.scanBalance(ctx, walletID, true)
}

func (r *WalletRepository) scanBalance(ctx context.Context, walletID uuid.UUID, forUpdate bool) (int64, error) {
	query := `SELECT balance FROM wallets WHERE id = $1`
	if forUpdate {
		query += ` FOR UPDATE`
	}

	var balance int64
	err := r.q(ctx).QueryRow(ctx, query, walletID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, walleterror.ErrWalletNotFound
		}
		return 0, fmt.Errorf("scan balance: %w", err)
	}

	return balance, nil
}

// UpdateBalance ...
func (r *WalletRepository) UpdateBalance(ctx context.Context, walletID uuid.UUID, newBalance int64) error {
	query := `UPDATE wallets SET balance = $1, updated_at = NOW() WHERE id = $2`

	_, err := r.q(ctx).Exec(ctx, query, newBalance, walletID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	return nil
}

// SaveOperation ...
func (r *WalletRepository) SaveOperation(ctx context.Context, op usecase.Operation) error {
	query := `
		INSERT INTO wallet_operations (id, wallet_id, operation, amount)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.q(ctx).Exec(ctx, query, op.ID, op.WalletID, op.Type, op.Amount)
	if err != nil {
		return fmt.Errorf("save operation: %w", err)
	}

	return nil
}
