// Package handler ...
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	walleterror "wallet/internal/error"
	"wallet/internal/port"
	"wallet/internal/port/middleware"
	"wallet/internal/validation"

	"github.com/google/uuid"
)

type WalletUsecase interface {
	// Deposit ...
	Deposit(ctx context.Context, in depositInput) error
	// Withdraw ...
	Withdraw(ctx context.Context, in withdrawInput) error
	// Balance ...
	Balance(ctx context.Context, walletID uuid.UUID) (int64, error)
}

// depositInput ...
type depositInput struct {
	WalletID uuid.UUID
	Amount   int64
}

// withdrawInput ...
type withdrawInput struct {
	WalletID uuid.UUID
	Amount   int64
}

type walletHandler struct {
	walletUsecase WalletUsecase
	server        *port.ServerAPI
}

// NewWalletHandler ...
func NewWalletHandler(walletUsecase WalletUsecase, server *port.ServerAPI) *walletHandler {
	return &walletHandler{
		walletUsecase: walletUsecase,
		server:        server,
	}
}

func (h *walletHandler) HandleOperation() http.HandlerFunc {
	const op = "walletHandler.HandleOperation"
	type req struct {
		ValletId      uuid.UUID `json:"valletId"`
		OperationType string    `json:"operationType"`
		Amount        int64     `json:"amount"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		log := h.server.Logger().With(
			slog.String("op", op),
			slog.String("requestID", middleware.GetRequestIDFromRequest(r)),
		)
		log.Info("processing operation")

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.With(
					slog.String("err", err.Error()),
				).Warn("body close with error")
			}
		}()

		req := &req{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			h.server.Error(w, r, op, err)
			return
		}

		t, err := validation.ValidationOperationType(req.OperationType)
		if err != nil {
			h.server.Error(w, r, op, err)
			return
		}
		if req.Amount <= 0 {
			h.server.Error(w, r, op, walleterror.ErrInvalidAmount)
			return
		}
		if req.ValletId == uuid.Nil {
			h.server.Error(w, r, op, walleterror.ErrInvalidValletID)
			return
		}

		ctx := r.Context()

		switch t {
		case validation.DepositType:
			dep := depositInput{
				WalletID: req.ValletId,
				Amount:   req.Amount,
			}
			if err := h.walletUsecase.Deposit(ctx, dep); err != nil {
				h.server.Error(w, r, op, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		case validation.WithdrawType:
			wdraw := withdrawInput{
				WalletID: req.ValletId,
				Amount:   req.Amount,
			}
			if err := h.walletUsecase.Withdraw(ctx, wdraw); err != nil {
				h.server.Error(w, r, op, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		default:
			h.server.Error(w, r, op, nil)
			return
		}
	}
}
