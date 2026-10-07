// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package spec

import "testing"

func TestTypedRelationshipsRejectLossyConversion(t *testing.T) {
	for _, relationship := range []string{"parent", "follows", "concurrent", "unknown"} {
		f := &FlowSpec{Graph: &GraphSpec{Nodes: []GraphNode{{ID: "a", Call: "svc.a"}, {ID: "b", Call: "svc.b"}}, Edges: []GraphEdge{{From: "a", To: "b", Relationship: relationship}}}}
		if _, err := ConvertGraphToFlow(f); err == nil {
			t.Fatalf("converted %s relationship without preserving its meaning", relationship)
		}
		err := f.Graph.ValidateGraphStructure()
		if (err != nil) != (relationship == "unknown") {
			t.Fatalf("relationship %s: %v", relationship, err)
		}
	}
}
