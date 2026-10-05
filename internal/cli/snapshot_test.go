// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/choreoatlas2025/cli/internal/baseline"
	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func snapshotHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

func TestSnapshotLoadedContractIdentity(t *testing.T) {
	dir, _, _ := correctnessFixture(t, []string{"A"}, "true")
	flowPath, servicePath, tracePath := filepath.Join(dir, "flow.yaml"), filepath.Join(dir, "svc.yaml"), filepath.Join(dir, "trace.json")
	wantFlow, wantService := snapshotHash(t, flowPath), snapshotHash(t, servicePath)
	files := input.NewSnapshot(nil)
	contract, err := loadAndValidateContract(flowPath, files)
	if err != nil {
		t.Fatal(err)
	}
	flow := contract.Flow
	tr, traceHash, err := loadTraceSnapshot(tracePath, files)
	if err != nil {
		t.Fatal(err)
	}
	results, passed := validate.ValidateAgainstTrace(flow, contract.Operations, tr, spec.DefaultValidationConfig())
	if !passed {
		t.Fatal(results)
	}
	config := spec.ValidationConfig{Semantic: true, Causality: "temporal", ToleranceMs: 50}
	provenance, err := executionIdentity(tr, traceHash, config, files)
	if err != nil {
		t.Fatal(err)
	}
	// Replace both contracts after parsing and validation. Recording must bind
	// the original bytes, even when the replacement has the same condition names.
	flow.Info.Title = "Replacement"
	correctnessWrite(t, flowPath, flow)
	flow.Info.Title = "Correctness"
	correctnessWrite(t, servicePath, spec.ServiceSpecFile{Service: "svc", Operations: []spec.ServiceOperation{{OperationId: "A", Postconditions: map[string]string{"status": "false"}}}})
	recorded, err := baseline.RecordBaseline(contract, results, provenance)
	if err != nil {
		t.Fatal(err)
	}
	if recorded.FlowHash != wantFlow || recorded.ServiceHashes["svc"] != wantService {
		t.Fatalf("validated input was rebound to replacement files: flow=%s service=%s; want %s %s", recorded.FlowHash, recorded.ServiceHashes["svc"], wantFlow, wantService)
	}
	fresh, err := loadAndValidateContract(flowPath, input.NewSnapshot(nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := baseline.ValidateCompatibility(recorded, fresh); err == nil {
		t.Fatal("a new invocation accepted replacement contracts as the recorded input")
	}
	for _, path := range []string{flowPath, servicePath, tracePath} {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	if err := baseline.ValidateCompatibility(recorded, contract); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotLoadedTraceIdentity(t *testing.T) {
	dir, _, spans := correctnessFixture(t, []string{"A"}, "true")
	path := filepath.Join(dir, "trace.json")
	want := snapshotHash(t, path)
	files := input.NewSnapshot(nil)
	tr, traceHash, err := loadTraceSnapshot(path, files)
	if err != nil {
		t.Fatal(err)
	}
	spans[0].Attributes["response.status"] = 500
	correctnessTrace(t, dir, spans)
	id, err := executionIdentity(tr, traceHash, spec.ValidationConfig{Semantic: true, Causality: "temporal", ToleranceMs: 50}, files)
	if err != nil {
		t.Fatal(err)
	}
	if id.TraceHash != want {
		t.Fatalf("loaded trace rebound to replacement bytes: got %s want %s", id.TraceHash, want)
	}
}
