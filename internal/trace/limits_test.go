// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package trace

import "testing"

func TestParseSpanBudgetAndDuplicateFields(t *testing.T) {
	for _, data := range []string{`{"spans":[{},{}]}`, `{"spans":null,"spans":[{},{}]}`} {
		if _, err := ParseWithMaxSpans([]byte(data), 1); err == nil {
			t.Fatal("span cap ignored")
		}
	}
	for _, data := range []string{`{"spans":null}`, `{"spans":null,"spans":[{}]}`, `{"spans":[{}],"spans":null}`} {
		if _, err := ParseWithMaxSpans([]byte(data), 1); err != nil {
			t.Fatal(err)
		}
	}
}
