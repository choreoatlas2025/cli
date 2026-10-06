//go:build darwin || linux

// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A FIFO lets the actual CLI open the original input, then replaces its path
// before EOF. Any subsequent open deterministically observes invalid bytes.
func snapshotReplaceBeforeEOF(t *testing.T, path string) <-chan error {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	done, stop := make(chan error, 1), make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		deadline := time.NewTimer(10 * time.Second)
		defer deadline.Stop()
		for {
			fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
			if errors.Is(err, syscall.ENXIO) {
				select {
				case <-stop:
					done <- errors.New("FIFO reader never opened")
					return
				case <-deadline.C:
					done <- errors.New("FIFO reader timeout")
					return
				case <-time.After(5 * time.Millisecond):
					continue
				}
			}
			if err != nil {
				done <- err
				return
			}
			writer := os.NewFile(uintptr(fd), path)
			_, err = writer.Write(data)
			if err == nil {
				err = os.Remove(path)
			}
			if err == nil {
				err = os.WriteFile(path, []byte("INVALID"), 0o644)
			}
			closeErr := writer.Close()
			if err == nil {
				err = closeErr
			}
			done <- err
			return
		}
	}()
	return done
}

func TestSnapshotCLIFileReplacement(t *testing.T) {
	for _, graph := range []bool{false, true} {
		for _, kind := range []string{"flow.yaml", "svc.yaml", "trace.json", "baseline.json"} {
			for _, format := range []string{"json", "html", "junit"} {
				mode := "flow"
				if graph {
					mode = "graph"
				}
				t.Run(mode+"/"+kind+"/"+format, func(t *testing.T) {
					dir, flow, _ := correctnessFixture(t, []string{"A"}, "true")
					if graph {
						correctnessGraph(flow, []string{"A"}, nil)
						correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
					}
					wantFlow, wantService, wantTrace := snapshotHash(t, filepath.Join(dir, "flow.yaml")), snapshotHash(t, filepath.Join(dir, "svc.yaml")), snapshotHash(t, filepath.Join(dir, "trace.json"))
					args := []string{"validate", "--flow", "flow.yaml", "--trace", "trace.json", "--report-format", format, "--report-out", "report." + format}
					wantBaseline := ""
					if kind == "baseline.json" {
						correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
						wantBaseline = snapshotHash(t, filepath.Join(dir, kind))
						args = append(args, "--baseline", "baseline.json")
					}
					done := snapshotReplaceBeforeEOF(t, filepath.Join(dir, kind))
					correctnessCommand(t, dir, 0, args...)
					if err := <-done; err != nil {
						t.Fatal(err)
					}
					data, err := os.ReadFile(filepath.Join(dir, "report."+format))
					if err != nil {
						t.Fatal(err)
					}
					var result struct {
						Success  bool `json:"success"`
						ExitCode int  `json:"exitCode"`
						Inputs   struct {
							Contract struct {
								FlowHash      string            `json:"flowHash"`
								ServiceHashes map[string]string `json:"serviceHashes"`
							} `json:"contract"`
							TraceHash    string `json:"traceHash"`
							BaselineHash string `json:"baselineHash"`
						} `json:"inputs"`
					}
					if format == "html" {
						_, payload, ok := strings.Cut(string(data), "window.FLOWREPORT = ")
						if !ok {
							t.Fatal("missing report payload")
						}
						data = []byte(strings.TrimSuffix(payload, ";</script>"))
					}
					if format == "junit" {
						var suite struct {
							Failures   int `xml:"failures,attr"`
							Properties []struct {
								Name  string `xml:"name,attr"`
								Value string `xml:"value,attr"`
							} `xml:"properties>property"`
						}
						if err := xml.Unmarshal(data, &suite); err != nil {
							t.Fatal(err)
						}
						if suite.Failures != 0 {
							t.Fatal("captured success failed JUnit")
						}
						found, foundExit := false, false
						for _, p := range suite.Properties {
							if p.Name == "result.inputs" {
								if err := json.Unmarshal([]byte(p.Value), &result.Inputs); err != nil {
									t.Fatal(err)
								}
								found = true
							}
							if p.Name == "result.exitCode" {
								foundExit = true
								if p.Value != "0" {
									t.Fatal("wrong JUnit exit")
								}
							}
						}
						if !found {
							t.Fatal("missing JUnit inputs")
						}
						if !foundExit {
							t.Fatal("missing JUnit exit")
						}
					} else {
						if err := json.Unmarshal(data, &result); err != nil {
							t.Fatal(err)
						}
						if !result.Success || result.ExitCode != 0 {
							t.Fatal("captured success failed report")
						}
					}
					id := result.Inputs
					if id.Contract.FlowHash != wantFlow || id.Contract.ServiceHashes["svc"] != wantService || id.TraceHash != wantTrace || id.BaselineHash != wantBaseline {
						t.Fatalf("%s replacement rebound report: %+v", kind, id)
					}
				})
			}
		}
	}
}

func TestSnapshotCLIRecordingAfterReplacement(t *testing.T) {
	for _, kind := range []string{"flow.yaml", "svc.yaml", "trace.json"} {
		t.Run(kind, func(t *testing.T) {
			dir, _, _ := correctnessFixture(t, []string{"A"}, "true")
			wantFlow, wantService, wantTrace := snapshotHash(t, filepath.Join(dir, "flow.yaml")), snapshotHash(t, filepath.Join(dir, "svc.yaml")), snapshotHash(t, filepath.Join(dir, "trace.json"))
			done := snapshotReplaceBeforeEOF(t, filepath.Join(dir, kind))
			correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(dir, "baseline.json"))
			if err != nil {
				t.Fatal(err)
			}
			var recorded struct {
				FlowHash      string            `json:"flowHash"`
				ServiceHashes map[string]string `json:"serviceHashes"`
				Provenance    struct {
					TraceHash string `json:"traceHash"`
				} `json:"provenance"`
			}
			if err := json.Unmarshal(data, &recorded); err != nil {
				t.Fatal(err)
			}
			if recorded.FlowHash != wantFlow || recorded.ServiceHashes["svc"] != wantService || recorded.Provenance.TraceHash != wantTrace {
				t.Fatalf("%s replacement rebound recording: %+v", kind, recorded)
			}
			// The next invocation must capture the changed input rather than reuse
			// a process-global cache from the successful run.
			correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
		})
	}
}
