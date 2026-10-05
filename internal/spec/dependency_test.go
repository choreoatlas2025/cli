// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"reflect"
	"testing"
)

func TestMixedDependencyNormalization(t *testing.T) {
	g := &GraphSpec{
		Nodes: []GraphNode{{ID: "A"}, {ID: "B", Depends: []string{"A", "A"}}, {ID: "C", Depends: []string{"B"}}},
		Edges: []GraphEdge{{From: "B", To: "C"}, {From: "B", To: "C"}, {From: "A", To: "B", Condition: "guard"}},
	}
	want := []GraphEdge{{From: "B", To: "C"}, {From: "A", To: "B", Condition: "guard"}, {From: "A", To: "B"}}
	for i := 0; i < 2; i++ {
		g.EnsureEdges()
		if !reflect.DeepEqual(g.Edges, want) {
			t.Fatalf("lost or duplicated dependency: %+v", g.Edges)
		}
	}
	if err := g.ValidateGraphStructure(); err != nil {
		t.Fatal(err)
	}
}
