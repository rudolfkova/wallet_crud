// Package txctx ...
package txctx

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ключ для хранения транзакции в контексте
type TxKey struct{}

// ExtractTx ...
func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(TxKey{}).(pgx.Tx)
	return tx, ok
}
