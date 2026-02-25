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

// querier — минимальный интерфейс для выполнения SQL.
// Его реализуют и *pgxpool.Pool, и pgx.Tx — это позволяет писать
// SQL-методы один раз, не дублируя if tx/else pool в каждом из них.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// WalletRepository реализует usecase.WalletRepository поверх PostgreSQL.
type WalletRepository struct {
	pool *pgxpool.Pool
}

// New создаёт WalletRepository.
func New(pool *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{pool: pool}
}

// q возвращает транзакцию из контекста если она есть,
// иначе — пул. Весь репозиторий использует только этот метод для запросов.
func (r *WalletRepository) q(ctx context.Context) querier {
	if tx, ok := txctx.ExtractTx(ctx); ok {
		return tx
	}
	return r.pool
}

// GetBalance возвращает текущий баланс кошелька без блокировки.
// Используется для чтения (GET /wallets/:id).
func (r *WalletRepository) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	return r.scanBalance(ctx, walletID, false)
}

// GetBalanceForUpdate возвращает баланс с блокировкой строки (SELECT ... FOR UPDATE).
// Должен вызываться только внутри транзакции — блокирует строку до COMMIT/ROLLBACK.
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

// UpdateBalance обновляет баланс кошелька.
// Должен вызываться внутри транзакции после GetBalanceForUpdate.
func (r *WalletRepository) UpdateBalance(ctx context.Context, walletID uuid.UUID, newBalance int64) error {
	query := `UPDATE wallets SET balance = $1, updated_at = NOW() WHERE id = $2`

	_, err := r.q(ctx).Exec(ctx, query, newBalance, walletID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	return nil
}

// SaveOperation сохраняет запись об операции в wallet_operations.
// Должен вызываться внутри транзакции — атомарно с UpdateBalance.
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
