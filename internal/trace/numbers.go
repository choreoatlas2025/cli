// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Normalize JSON numbers before CEL adapts them. Integral values retain their
// exact signed/unsigned 64-bit value. Decimal/exponent spellings retain their
// float64 arithmetic semantics; integral values in those spellings must be
// representable without rounding. Unsupported ranges fail explicitly.
func normalizeNumber(number json.Number) (any, error) {
	text := number.String()
	if !strings.ContainsAny(text, ".eE") {
		if integer, err := number.Int64(); err == nil {
			return integer, nil
		}
		if integer, err := strconv.ParseUint(text, 10, 64); err == nil {
			return integer, nil
		}
		return nil, fmt.Errorf("integer outside signed/unsigned 64-bit range: %s", text)
	}
	// Checking float range first also bounds exponent expansion in big.Rat.
	floating, err := number.Float64()
	if err != nil {
		return nil, fmt.Errorf("unsupported numeric range %s: %w", text, err)
	}
	if floating == 0 {
		mantissa, _, _ := strings.Cut(strings.ToLower(text), "e")
		if strings.ContainsAny(mantissa, "123456789") {
			return nil, fmt.Errorf("numeric underflow: %s", text)
		}
		return floating, nil
	}
	exact, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, fmt.Errorf("invalid JSON number: %s", text)
	}
	if exact.IsInt() && exact.Cmp(new(big.Rat).SetFloat64(floating)) != 0 {
		return nil, fmt.Errorf("floating-point spelling loses integer precision: %s; use an integer literal within 64-bit range", text)
	}
	return floating, nil
}

func normalizeNumbers(value any) (any, error) {
	switch current := value.(type) {
	case json.Number:
		return normalizeNumber(current)
	case map[string]any:
		for key, child := range current {
			converted, err := normalizeNumbers(child)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", key, err)
			}
			current[key] = converted
		}
	case []any:
		for i, child := range current {
			converted, err := normalizeNumbers(child)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			current[i] = converted
		}
	}
	return value, nil
}
