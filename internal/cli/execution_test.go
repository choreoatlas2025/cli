// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestArchitectureUnavailableToolIdentity(t *testing.T) {
	tr := &trace.Trace{}
	failure := errors.New("executable path unavailable")
	for _, tc := range []struct {
		executable func() (string, error)
		reader     func(string) ([]byte, error)
		want       error
	}{
		{func() (string, error) { return "", failure }, nil, failure},
		{func() (string, error) { return "/missing/tool", nil }, func(string) ([]byte, error) { return nil, os.ErrPermission }, os.ErrPermission},
	} {
		id, err := executionIdentityWith(tr, input.HashBytes([]byte("trace")), spec.DefaultValidationConfig(), input.NewSnapshot(tc.reader), tc.executable)
		if !errors.Is(err, tc.want) || !strings.Contains(err.Error(), "evidence incomplete") || id.ValidatorHash != "" {
			t.Fatal("missing identity was hidden", id, err)
		}
		if err := id.Validate(); err == nil {
			t.Fatal("incomplete tool identity allowed baseline")
		}
	}
}
