// Package sqlstore ...
package sqlstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	txctx "wallet/internal/driver"
)

// Store хранит пул соединений и реализует TxManager.
type Store struct {
	pool *pgxpool.Pool
}

// New создаёт Store и проверяет соединение с БД.
func New(ctx context.Context, connString string) (*Store, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Store{pool: pool}, nil
}

// Close закрывает пул соединений.
func (s *Store) Close() {
	s.pool.Close()
}

// Pool возвращает пул соединений — нужен для создания репозитория.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// RunInTx открывает транзакцию, кладёт её в контекст и вызывает fn.
// Если fn вернула ошибку — откатывает, иначе коммитит.
func (s *Store) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// кладём транзакцию в контекст, чтобы репозиторий мог её достать
	ctx = context.WithValue(ctx, txctx.TxKey{}, tx)

	if err := fn(ctx); err != nil {
		// откат — логируем его ошибку, но возвращаем оригинальную
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %w (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
