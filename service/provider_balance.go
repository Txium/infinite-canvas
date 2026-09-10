package service

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
)

// Retain the original decimal string; float validation is not used for storing
// money. Missing, malformed or non-finite values must never become a zero balance.
func providerBalanceAmount(raw json.RawMessage) (string, error) {
	value := strings.TrimSpace(string(raw))
	if strings.HasPrefix(value, "\"") {
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", errors.New("上游余额格式无效")
		}
		value = strings.TrimSpace(value)
	}
	var number json.Number
	if len(value) == 0 || len(value) > 128 || json.Unmarshal([]byte(value), &number) != nil || number == "" {
		return "", errors.New("上游未返回有效余额，保留最近一次核实值")
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return "", errors.New("上游余额数值无效")
	}
	return value, nil
}
