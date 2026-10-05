// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/schemas"
)

func TestContractSnapshotSchemaParseIdentityAndSharedService(t *testing.T) {
	dir := t.TempDir()
	flowPath, servicePath := filepath.Join(dir, "flow.yaml"), filepath.Join(dir, "svc.yaml")
	flowData := []byte("info:\n  title: Original\nservices:\n  one: {spec: svc.yaml}\n  two: {spec: ./svc.yaml}\nflow:\n  - step: Check\n    call: one.check\n")
	serviceData := []byte("service: svc\noperations:\n  - operationId: check\n    postconditions: {ok: 'true'}\n")
	for path, data := range map[string][]byte{flowPath: flowData, servicePath: serviceData} {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reads := map[string]int{}
	files := input.NewSnapshot(func(path string) ([]byte, error) {
		reads[path]++
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		// Change to schema-invalid input before the caller receives the bytes.
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			return nil, err
		}
		return data, nil
	})
	contract, err := LoadContractSnapshot(flowPath, files, schemas.FS)
	if err != nil {
		t.Fatal(err)
	}
	if reads[flowPath] != 1 || reads[servicePath] != 1 {
		t.Fatalf("input reopened: %+v", reads)
	}
	identity := contract.Identity()
	if contract.Flow.Info.Title != "Original" || identity.FlowHash != input.HashBytes(flowData) || identity.ServiceHashes["one"] != input.HashBytes(serviceData) || identity.ServiceHashes["two"] != identity.ServiceHashes["one"] {
		t.Fatalf("mixed capture: %+v", identity)
	}
	for _, alias := range []string{"one", "two"} {
		if contract.Operations[alias]["check"].Postconditions["ok"] != "true" {
			t.Fatal("parsed replacement operation")
		}
	}
	identity.ServiceHashes["one"] = "tampered"
	if contract.Identity().ServiceHashes["one"] != input.HashBytes(serviceData) {
		t.Fatal("identity map exposed mutable capture")
	}
}
