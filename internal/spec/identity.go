// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"fmt"
	"io/fs"
	"maps"
	"sort"

	"github.com/choreoatlas2025/cli/internal/input"
)

// ContractIdentity binds all contract files, including service conditions.
type ContractIdentity struct {
	FlowID        string            `json:"flowId"`
	FlowHash      string            `json:"flowHash"`
	ServiceHashes map[string]string `json:"serviceHashes"`
}

// ContractSnapshot binds parsed contracts, operations and identity to the same
// captured input bytes. Baseline and report consumers never reopen the files.
type ContractSnapshot struct {
	Flow       *FlowSpec
	Operations map[string]map[string]ServiceOperation
	identity   ContractIdentity
}

func (c *ContractSnapshot) Identity() ContractIdentity {
	id := c.identity
	id.ServiceHashes = maps.Clone(id.ServiceHashes)
	return id
}

// A nil schema FS explicitly skips schema checks, not parsing or duplicate IDs.
func LoadContractSnapshot(path string, files *input.Snapshot, schemaFS fs.FS) (*ContractSnapshot, error) {
	file, err := files.Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read FlowSpec: %w", err)
	}
	data := file.Bytes()
	if schemaFS != nil {
		if err := ValidateYAMLBytesWithSchemaFS(data, schemaFS, "flowspec.schema.json"); err != nil {
			return nil, fmt.Errorf("invalid FlowSpec: %w", err)
		}
	}
	flow, err := ParseFlowSpec(data)
	if err != nil {
		return nil, err
	}
	c := &ContractSnapshot{Flow: flow, Operations: map[string]map[string]ServiceOperation{}, identity: ContractIdentity{FlowID: flow.Info.Title, FlowHash: file.Hash(), ServiceHashes: map[string]string{}}}
	aliases := make([]string, 0, len(flow.Services))
	for alias := range flow.Services {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		serviceFile, err := files.Read(ResolvePath(path, flow.Services[alias].Spec))
		if err != nil {
			return nil, fmt.Errorf("failed to load service %q spec: %w", alias, err)
		}
		serviceData := serviceFile.Bytes()
		if schemaFS != nil {
			if err := ValidateYAMLBytesWithSchemaFS(serviceData, schemaFS, "servicespec.schema.json"); err != nil {
				return nil, fmt.Errorf("invalid ServiceSpec %s: %w", alias, err)
			}
		}
		service, err := ParseServiceSpec(serviceData)
		if err != nil {
			return nil, fmt.Errorf("failed to load service %q spec: %w", alias, err)
		}
		operations := map[string]ServiceOperation{}
		for _, op := range service.Operations {
			operations[op.OperationId] = op
		}
		c.Operations[alias] = operations
		c.identity.ServiceHashes[alias] = serviceFile.Hash()
	}
	return c, nil
}
