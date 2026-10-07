// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"reflect"
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestDiscoveryDoesNotTurnRetryOutcomesIntoLastValueRule(t *testing.T) {
	failed := trace.Span{Attributes: map[string]any{"http.method": "GET", "http.status_code": 500}}
	successful := trace.Span{Attributes: map[string]any{"http.method": "GET", "http.status_code": 200}}
	a := generateServiceOperation("get", []trace.Span{failed, successful})
	b := generateServiceOperation("get", []trace.Span{successful, failed})
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("trace order changed sampled requirements: %+v / %+v", a, b)
	}
	if len(a.Postconditions) != 0 || len(a.Preconditions) != 1 || !strings.Contains(a.Description, "varying") {
		t.Fatalf("retry variation hidden or last status imposed on all calls: %+v", a)
	}
	fixed := generateServiceOperation("get", []trace.Span{successful, successful})
	if len(fixed.Postconditions) != 1 {
		t.Fatal("identical successful outcomes lost their sampled condition")
	}
	missing := generateServiceOperation("get", []trace.Span{successful, {Attributes: map[string]any{}}})
	if len(missing.Preconditions) != 0 || len(missing.Postconditions) != 0 {
		t.Fatal("absent attributes became universal conditions")
	}
}
