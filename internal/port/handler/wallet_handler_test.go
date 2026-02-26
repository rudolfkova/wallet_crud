// Package handler_test ...
package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	walleterror "wallet/internal/error"
	"wallet/internal/mocks"
	"wallet/internal/model"
	"wallet/internal/port"
	"wallet/internal/port/handler"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServer() *port.ServerAPI {
	return port.NewServer(newDiscardLogger())
}

func sendRequest(t *testing.T, handler http.HandlerFunc, method, url string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&reqBody).Encode(body))
	}

	req := httptest.NewRequest(method, url, &reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder, dst any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(rr.Body).Decode(dst))
}

// --- HandleGetBalance ---

func TestHandleGetBalance_Success(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	walletID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	expectedBalance := int64(1000)

	uc.
		On("Balance", mock.Anything, walletID).
		Return(expectedBalance, nil)

	h := handler.NewWalletHandler(uc, newTestServer())

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/wallets/{id}", h.HandleGetBalance())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+walletID.String(), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp model.BalanceResponse
	decodeBody(t, rr, &resp)
	assert.Equal(t, walletID, resp.WalletID)
	assert.Equal(t, expectedBalance, resp.Balance)
	uc.AssertExpectations(t)
}

func TestHandleGetBalance_InvalidUUID(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	h := handler.NewWalletHandler(uc, newTestServer())

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/wallets/{id}", h.HandleGetBalance())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	uc.AssertNotCalled(t, "Balance")
}

func TestHandleGetBalance_WalletNotFound(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	walletID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	uc.
		On("Balance", mock.Anything, walletID).
		Return(int64(0), walleterror.ErrWalletNotFound)

	h := handler.NewWalletHandler(uc, newTestServer())

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/wallets/{id}", h.HandleGetBalance())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+walletID.String(), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)

	var resp port.ErrorResponse
	decodeBody(t, rr, &resp)
	assert.Equal(t, "NOT_FOUND", resp.Code)
	uc.AssertExpectations(t)
}

// --- HandleOperation / DEPOSIT ---

func TestHandleOperation_Deposit_Success(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	walletID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	uc.
		On("Deposit", mock.Anything, model.DepositInput{
			WalletID: walletID,
			Amount:   500,
		}).
		Return(nil)

	h := handler.NewWalletHandler(uc, newTestServer())
	rr := sendRequest(t, h.HandleOperation(), http.MethodPost, "/api/v1/wallet", map[string]any{
		"valletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        500,
	})

	assert.Equal(t, http.StatusOK, rr.Code)
	uc.AssertExpectations(t)
}

func TestHandleOperation_Deposit_WalletNotFound(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	walletID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	uc.
		On("Deposit", mock.Anything, model.DepositInput{
			WalletID: walletID,
			Amount:   500,
		}).
		Return(walleterror.ErrWalletNotFound)

	h := handler.NewWalletHandler(uc, newTestServer())
	rr := sendRequest(t, h.HandleOperation(), http.MethodPost, "/api/v1/wallet", map[string]any{
		"valletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        500,
	})

	assert.Equal(t, http.StatusNotFound, rr.Code)
	uc.AssertExpectations(t)
}

// --- HandleOperation / WITHDRAW ---

func TestHandleOperation_Withdraw_Success(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	walletID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	uc.
		On("Withdraw", mock.Anything, model.WithdrawInput{
			WalletID: walletID,
			Amount:   200,
		}).
		Return(nil)

	h := handler.NewWalletHandler(uc, newTestServer())
	rr := sendRequest(t, h.HandleOperation(), http.MethodPost, "/api/v1/wallet", map[string]any{
		"valletId":      walletID,
		"operationType": "WITHDRAW",
		"amount":        200,
	})

	assert.Equal(t, http.StatusOK, rr.Code)
	uc.AssertExpectations(t)
}

func TestHandleOperation_Withdraw_InsufficientFunds(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	walletID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	uc.
		On("Withdraw", mock.Anything, model.WithdrawInput{
			WalletID: walletID,
			Amount:   99999,
		}).
		Return(walleterror.ErrInsufficientFunds)

	h := handler.NewWalletHandler(uc, newTestServer())
	rr := sendRequest(t, h.HandleOperation(), http.MethodPost, "/api/v1/wallet", map[string]any{
		"valletId":      walletID,
		"operationType": "WITHDRAW",
		"amount":        99999,
	})

	assert.Equal(t, http.StatusConflict, rr.Code)

	var resp port.ErrorResponse
	decodeBody(t, rr, &resp)
	assert.Equal(t, "CONFLICT", resp.Code)
	uc.AssertExpectations(t)
}

// --- HandleOperation / Валидация ---

func TestHandleOperation_InvalidOperationType(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	walletID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	h := handler.NewWalletHandler(uc, newTestServer())
	rr := sendRequest(t, h.HandleOperation(), http.MethodPost, "/api/v1/wallet", map[string]any{
		"valletId":      walletID,
		"operationType": "INVALID",
		"amount":        100,
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	uc.AssertNotCalled(t, "Deposit")
	uc.AssertNotCalled(t, "Withdraw")
}

func TestHandleOperation_InvalidAmount(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	walletID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	h := handler.NewWalletHandler(uc, newTestServer())
	rr := sendRequest(t, h.HandleOperation(), http.MethodPost, "/api/v1/wallet", map[string]any{
		"valletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        -100,
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	uc.AssertNotCalled(t, "Deposit")
}

func TestHandleOperation_NilWalletID(t *testing.T) {
	uc := new(mocks.WalletUsecase)

	h := handler.NewWalletHandler(uc, newTestServer())
	rr := sendRequest(t, h.HandleOperation(), http.MethodPost, "/api/v1/wallet", map[string]any{
		"valletId":      uuid.Nil,
		"operationType": "DEPOSIT",
		"amount":        100,
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	uc.AssertNotCalled(t, "Deposit")
}

func TestHandleOperation_InvalidBody(t *testing.T) {
	uc := new(mocks.WalletUsecase)
	h := handler.NewWalletHandler(uc, newTestServer())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.HandleOperation().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	uc.AssertNotCalled(t, "Deposit")
	uc.AssertNotCalled(t, "Withdraw")
}
