// Package validation ...
package validation

import (
	"strings"
	walleterror "wallet/internal/error"
)

var (
	// DepositType ...
	DepositType = "DEPOSIT"
	WithdrawType    = "WITHDRAW"
)

func ValidationOperationType(t string) (string, error) {
	tStr := strings.TrimSpace(t)
	switch tStr {
	case "":
		return "", walleterror.ErrTypeNotSpecified
	case DepositType:
		return tStr, nil
	case WithdrawType:
		return tStr, nil
	default:
		return "", walleterror.ErrInvalidOperationType

	}
}
