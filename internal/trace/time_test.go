// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestTimePresenceRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		body       string
		start, end bool
	}{
		{`{"spans":[{"name":"a","service":"svc"}]}`, false, false},
		{`{"spans":[{"name":"a","service":"svc","startNanos":0}]}`, true, false},
		{`{"spans":[{"name":"a","service":"svc","endNanos":0}]}`, false, true},
		{`{"spans":[{"name":"a","service":"svc","startNanos":0,"endNanos":0}]}`, true, true},
	} {
		tr, err := Parse([]byte(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(tr)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("startNanos")) != tc.start || bytes.Contains(data, []byte("endNanos")) != tc.end {
			t.Fatalf("round trip invented or lost timestamp: %s", data)
		}
		again, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		span := again.Spans[0]
		if span.HasStart() != tc.start || span.HasEnd() != tc.end {
			t.Fatal("lost presence on second parse")
		}
		if err := ValidateTimestamps(again.Spans, true); (err == nil) != (tc.start && tc.end) {
			t.Fatalf("qualification mismatch: %v", err)
		}
	}
	// Native Go callers can explicitly supply zero without a decoder sentinel.
	data, err := json.Marshal(Span{Name: "a", Service: "svc"})
	if err != nil || !bytes.Contains(data, []byte(`"startNanos":0`)) || !bytes.Contains(data, []byte(`"endNanos":0`)) {
		t.Fatalf("programmatic zero timestamps lost: %s, %v", data, err)
	}
}

func TestTimeInvalidTimestampTypes(t *testing.T) {
	for _, field := range []string{"startNanos", "endNanos"} {
		for _, value := range []string{"null", `"0"`, "0.5", "1.0", "false", "9223372036854775808"} {
			if _, err := Parse([]byte(`{"spans":[{"` + field + `":` + value + `}]}`)); err == nil {
				t.Errorf("accepted invalid timestamp %s=%s", field, value)
			}
		}
	}
}
