// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
    "regexp"
    "strings"
)

var nonWord = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// NormalizeServiceAlias converts a service name into a stable lowerCamel alias.
func NormalizeServiceAlias(name string) string {
    if name == "" { return "service" }
    raw := nonWord.Split(name, -1)
    // collect non-empty tokens
    tokens := make([]string, 0, len(raw))
    for _, p := range raw {
        p = strings.TrimSpace(p)
        if p != "" { tokens = append(tokens, p) }
    }
    var b strings.Builder
    for i, p := range tokens {
        if i == 0 {
            b.WriteString(strings.ToLower(p[:1]))
            if len(p) > 1 { b.WriteString(p[1:]) }
        } else {
            b.WriteString(strings.ToUpper(p[:1]))
            if len(p) > 1 { b.WriteString(p[1:]) }
        }
    }
    out := b.String()
    if out == "" { return "service" }
    return out
}

// NormalizeOperationID turns raw names (HTTP/RPC/custom) into a stable operationId.
func NormalizeOperationID(raw string) string {
    s := strings.TrimSpace(raw)
    if s == "" { return "op" }

    up := strings.ToUpper(s)
    methods := []string{"GET ", "POST ", "PUT ", "PATCH ", "DELETE ", "HEAD ", "OPTIONS "}
    for _, m := range methods {
        if strings.HasPrefix(up, m) {
            method := strings.ToLower(strings.TrimSpace(m))
            path := strings.TrimSpace(s[len(strings.TrimSpace(m))+1:])
            segs := strings.Split(path, "/")
            parts := []string{method}
            for _, seg := range segs {
                if seg == "" { continue }
                seg = strings.TrimSpace(seg)
                if seg == "" { continue }
                if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
                    name := strings.Trim(seg, "{}")
                    parts = append(parts, "By"+toPascalCase(name))
                } else if strings.HasPrefix(seg, ":") {
                    name := strings.TrimPrefix(seg, ":")
                    parts = append(parts, "By"+toPascalCase(name))
                } else {
                    parts = append(parts, toPascalCase(seg))
                }
            }
            return strings.Join(parts, "")
        }
    }

    if strings.Contains(s, ".") || strings.Contains(s, "/") {
        last := s
        if i := strings.LastIndexAny(s, "/."); i >= 0 && i+1 < len(s) {
            last = s[i+1:]
        }
        return toLowerCamelCase(nonWord.ReplaceAllString(last, " "))
    }
    return toLowerCamelCase(nonWord.ReplaceAllString(s, " "))
}

func toPascalCase(s string) string {
    s = strings.TrimSpace(s)
    if s == "" { return "" }
    parts := strings.Fields(s)
    var b strings.Builder
    for _, p := range parts {
        if p == "" { continue }
        b.WriteString(strings.ToUpper(p[:1]))
        if len(p) > 1 { b.WriteString(p[1:]) }
    }
    return b.String()
}

func toLowerCamelCase(s string) string {
    s = strings.TrimSpace(s)
    if s == "" { return "" }
    parts := strings.Fields(s)
    var b strings.Builder
    for i, p := range parts {
        if p == "" { continue }
        if i == 0 { b.WriteString(strings.ToLower(p[:1])) } else { b.WriteString(strings.ToUpper(p[:1])) }
        if len(p) > 1 { b.WriteString(p[1:]) }
    }
    out := b.String()
    if out == "" { return "id" }
    return out
}
