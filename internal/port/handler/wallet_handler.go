// Package handler ...
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	walleterror "wallet/internal/error"
	"wallet/internal/model"
	"wallet/internal/port"
	"wallet/internal/port/middleware"
	"wallet/internal/validation"

	"github.com/google/uuid"
)

type WalletUsecase interface {
	// Deposit ...
	Deposit(ctx context.Context, in model.DepositInput) error
	// Withdraw ...
	Withdraw(ctx context.Context, in model.WithdrawInput) error
	// Balance ...
	Balance(ctx context.Context, walletID uuid.UUID) (int64, error)
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
			dep := model.DepositInput{
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
			wdraw := model.WithdrawInput{
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

func (h *walletHandler) HandleGetBalance() http.HandlerFunc {
	const op = "walletHandler.HandleGetBalance"
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		walletID, err := uuid.Parse(idStr)
		if err != nil {
			h.server.Error(w, r, op, walleterror.ErrInvalidValletID)
			return
		}
		if walletID == uuid.Nil {
			h.server.Error(w, r, op, walleterror.ErrInvalidValletID)
			return
		}

		log := h.server.Logger().With(
			slog.String("op", op),
			slog.String("requestID", middleware.GetRequestIDFromRequest(r)),
			slog.String("walletID", walletID.String()),
		)

		log.Info("get wallet")

		ctx := r.Context()

		balance, err := h.walletUsecase.Balance(ctx, walletID)
		if err != nil {
			h.server.Error(w, r, op, err)
			return
		}

		balanceDTO := model.BalanceResponse{
			WalletID: walletID,
			Balance:  balance,
		}

		h.server.Respond(w, r, http.StatusOK, balanceDTO)
	}
}
