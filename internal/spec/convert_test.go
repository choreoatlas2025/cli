// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"reflect"
	"testing"
)

func TestConversionPreservesUnconditionalChain(t *testing.T) {
	flow := &FlowSpec{
		Info:     FlowInfo{Title: "Chain", Version: "1", Description: "Description"},
		Services: map[string]ServiceBinding{"svc": {Spec: "svc.yaml"}},
		Graph: &GraphSpec{
			Nodes: []GraphNode{
				{ID: "last", Call: "svc.last", Depends: []string{"first"}, Input: map[string]any{"id": "${id}"}, Output: map[string]string{"done": "response.body"}, Meta: map[string]any{"note": "keep"}},
				{ID: "first", Call: "svc.first", Output: map[string]string{"id": "response.body.id"}},
			},
		},
	}
	converted, err := ConvertGraphToFlow(flow)
	if err != nil {
		t.Fatal(err)
	}
	if converted.Graph != nil || len(converted.Flow) != 2 || converted.Flow[0].Step != "first" || converted.Flow[1].Step != "last" {
		t.Fatalf("invalid conversion: %+v", converted)
	}
	if !reflect.DeepEqual(converted.Info, flow.Info) || !reflect.DeepEqual(converted.Flow[1].Input, flow.Graph.Nodes[0].Input) || !reflect.DeepEqual(converted.Flow[1].Output, flow.Graph.Nodes[0].Output) || !reflect.DeepEqual(converted.Flow[1].Meta, flow.Graph.Nodes[0].Meta) {
		t.Fatal("conversion discarded contract content")
	}
	converted.Services["svc"] = ServiceBinding{Spec: "different.yaml"}
	if flow.Services["svc"].Spec != "svc.yaml" {
		t.Fatal("conversion mutated input bindings")
	}
}

func TestConversionRejectsUnrepresentableGraphs(t *testing.T) {
	for _, graph := range []*GraphSpec{
		{},
		{Nodes: []GraphNode{{ID: "A", Call: "svc.A"}, {ID: "B", Call: "svc.B"}}},
		{Nodes: []GraphNode{{ID: "A", Call: "svc.A"}, {ID: "B", Call: "svc.B"}}, Edges: []GraphEdge{{From: "A", To: "B", Condition: "true"}}},
	} {
		if _, err := ConvertGraphToFlow(&FlowSpec{Graph: graph}); err == nil {
			t.Errorf("accepted unsupported graph: %+v", graph)
		}
	}
}
