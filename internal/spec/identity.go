// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// ContractIdentity binds all contract files, including service conditions.
type ContractIdentity struct {
	FlowID        string            `json:"flowId"`
	FlowHash      string            `json:"flowHash"`
	ServiceHashes map[string]string `json:"serviceHashes"`
}

func HashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(b)), nil
}

func IdentifyContract(flow *FlowSpec, flowPath string) (ContractIdentity, error) {
	id := ContractIdentity{FlowID: flow.Info.Title, ServiceHashes: map[string]string{}}
	var err error
	id.FlowHash, err = HashFile(flowPath)
	if err != nil {
		return id, fmt.Errorf("failed to hash FlowSpec: %w", err)
	}
	for alias, binding := range flow.Services {
		hash, err := HashFile(ResolvePath(flowPath, binding.Spec))
		if err != nil {
			return id, fmt.Errorf("failed to hash ServiceSpec %s: %w", alias, err)
		}
		id.ServiceHashes[alias] = hash
	}
	return id, nil
}
