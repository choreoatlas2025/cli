// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"reflect"
	"testing"
)

func TestParseExactAttributeNumbers(t *testing.T) {
	tr, err := Parse([]byte(`{"spans":[{"attributes":{"body":{"ids":[9007199254740992,9007199254740993,-9007199254740993,9223372036854775807,18446744073709551615,9007199254740992.0,9.007199254740992e15,1.25,0e999999999]},"text":"9007199254740993"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []any{int64(9007199254740992), int64(9007199254740993), int64(-9007199254740993), int64(9223372036854775807), uint64(18446744073709551615), float64(9007199254740992), float64(9007199254740992), float64(1.25), float64(0)}
	got := tr.Spans[0].Attributes["body"].(map[string]any)["ids"]
	if !reflect.DeepEqual(got, want) || tr.Spans[0].Attributes["text"] != "9007199254740993" {
		t.Fatalf("numbers lost identity or type: %+v", got)
	}
}

func TestParseRejectsNumericRangeAndTrailingData(t *testing.T) {
	for _, number := range []string{"18446744073709551616", "-9223372036854775809", "1e400", "1e-999", "9007199254740993.0", "9.007199254740993e15"} {
		if _, err := Parse([]byte(`{"spans":[{"attributes":{"nested":[` + number + `]}}]}`)); err == nil {
			t.Errorf("unsupported number silently rounded: %s", number)
		}
	}
	for _, trailing := range []string{"{}", "garbage"} {
		if _, err := Parse([]byte(`{"spans":[]} ` + trailing)); err == nil {
			t.Errorf("accepted trailing document: %s", trailing)
		}
	}
}
