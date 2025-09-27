// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

// ConvertGraphToFlow converts a GraphSpec into flow format. It performs a
// topo-order traversal and when a node has multiple successors and each
// successor has a single predecessor, it folds those successors into a
// Parallel block under the parent node.
func ConvertGraphToFlow(fs *FlowSpec) *FlowSpec {
    if fs == nil || fs.Graph == nil { return fs }
    g := fs.Graph
    g.EnsureEdges()
    // build in-degree and adjacency
    inDeg := map[string]int{}
    adj := map[string][]string{}
    nodes := map[string]GraphNode{}
    for _, n := range g.Nodes {
        nodes[n.ID] = n
        inDeg[n.ID] = 0
    }
    for _, e := range g.Edges {
        adj[e.From] = append(adj[e.From], e.To)
        inDeg[e.To]++
    }
    // keep original in-degrees for parallel eligibility check
    origIn := map[string]int{}
    for k, v := range inDeg { origIn[k] = v }

    // Kahn topo with visited to avoid duplications when folding
    visited := map[string]bool{}
    queue := []string{}
    for id, d := range inDeg { if d == 0 { queue = append(queue, id) } }

    var flow []FlowStep
    for len(queue) > 0 {
        id := queue[0]
        queue = queue[1:]
        if visited[id] { continue }
        visited[id] = true
        n := nodes[id]

        succ := adj[id]
        // candidates for parallel: successors whose original in-degree == 1
        parCandidates := []string{}
        for _, s := range succ { if origIn[s] == 1 { parCandidates = append(parCandidates, s) } }

        if len(parCandidates) > 1 {
            parent := nodeToStep(n)
            // children are folded as parallel steps
            for _, s := range parCandidates {
                child := nodeToStep(nodes[s])
                parent.Parallel = append(parent.Parallel, child)
                visited[s] = true
                // decrease in-degree of their successors
                for _, ns := range adj[s] {
                    if inDeg[ns] > 0 { inDeg[ns]-- }
                    if inDeg[ns] == 0 { queue = append(queue, ns) }
                }
            }
            flow = append(flow, parent)
        } else {
            // regular sequential step
            flow = append(flow, nodeToStep(n))
            for _, s := range succ {
                if inDeg[s] > 0 { inDeg[s]-- }
                if inDeg[s] == 0 { queue = append(queue, s) }
            }
        }
    }

    out := &FlowSpec{ Info: fs.Info, Services: fs.Services, Flow: flow }
    return out
}

func nodeToStep(n GraphNode) FlowStep {
    return FlowStep{ Step: n.ID, Call: n.Call, Input: n.Input, Output: n.Output, Meta: n.Meta }
}

// WriteFlowSpec writes FlowSpec as YAML to file
func WriteFlowSpec(path string, fs *FlowSpec) error {
    b, err := yaml.Marshal(fs)
    if err != nil { return fmt.Errorf("failed to marshal flowspec: %w", err) }
    return os.WriteFile(path, b, 0644)
}

