// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
    "regexp"
    "strings"

    "github.com/choreoatlas2025/cli/internal/trace"
)

var httpNameRe = regexp.MustCompile(`^([A-Z]+)\s+(/.*)$`)
var nonAlnumRe = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// ComputeOperationID derives a deterministic operationId from a span
// Priority: HTTP(method+route) -> RPC(method) -> sanitized span.Name
func ComputeOperationID(span trace.Span) string {
    if m, p, ok := extractHTTP(span); ok {
        return normalizeHTTP(m, p)
    }
    if rm := getStringAttr(span, "rpc.method"); rm != "" {
        return lowerCamel(pascalize(rm))
    }
    // Fallback: sanitize span name
    return lowerCamel(pascalize(span.Name))
}

func extractHTTP(span trace.Span) (method string, path string, ok bool) {
    m := strings.ToUpper(getStringAttr(span, "http.method"))
    // Prefer http.route; otherwise derive from http.target or http.url
    route := getStringAttr(span, "http.route")
    if route == "" {
        route = getStringAttr(span, "http.target")
    }
    if route == "" {
        url := getStringAttr(span, "http.url")
        route = extractPathFromURL(url)
    }

    if m != "" && route != "" {
        return m, route, true
    }
    // Try parse from span name like "GET /health"
    if matches := httpNameRe.FindStringSubmatch(strings.TrimSpace(span.Name)); len(matches) == 3 {
        return strings.ToUpper(matches[1]), matches[2], true
    }
    return "", "", false
}

func extractPathFromURL(url string) string {
    if url == "" {
        return ""
    }
    // If it already looks like a path
    if strings.HasPrefix(url, "/") {
        return url
    }
    // crude split to handle http(s)://host[:port]/path?query#frag
    // find first '/' after scheme://host
    ix := strings.Index(url, "://")
    if ix >= 0 {
        rest := url[ix+3:]
        if j := strings.Index(rest, "/"); j >= 0 {
            rest = rest[j:]
            // strip query/fragment
            if k := strings.IndexAny(rest, "?#"); k >= 0 {
                rest = rest[:k]
            }
            return rest
        }
        return "/"
    }
    // Not a full URL; return as-is
    return url
}

func normalizeHTTP(method, route string) string {
    method = strings.ToLower(strings.TrimSpace(method))
    // normalize path tokens
    path := route
    if path == "" {
        path = "/"
    }
    // collapse multiple slashes
    for strings.Contains(path, "//") {
        path = strings.ReplaceAll(path, "//", "/")
    }
    tokens := strings.Split(path, "/")
    var parts []string
    var byParts []string
    for _, tok := range tokens {
        if tok == "" { // leading/trailing slash
            continue
        }
        // path parameter in {param} or :param style
        if strings.HasPrefix(tok, "{") && strings.HasSuffix(tok, "}") {
            name := tok[1 : len(tok)-1]
            byParts = append(byParts, "By"+pascalize(name))
            continue
        }
        if strings.HasPrefix(tok, ":") && len(tok) > 1 {
            name := tok[1:]
            byParts = append(byParts, "By"+pascalize(name))
            continue
        }
        // regular segment
        parts = append(parts, pascalize(tok))
    }
    if len(parts) == 0 {
        parts = []string{"Root"}
    }
    op := method + strings.Join(parts, "") + strings.Join(byParts, "")
    // ensure only [A-Za-z0-9_]
    op = nonAlnumRe.ReplaceAllString(op, "")
    if op == "" {
        op = method + "Op"
    }
    return op
}

func pascalize(s string) string {
    // replace separators with space, then upper-case initials
    repl := strings.NewReplacer("-", " ", "_", " ", ".", " ")
    s = repl.Replace(s)
    fields := strings.Fields(s)
    for i, f := range fields {
        if f == "" { continue }
        fields[i] = strings.ToUpper(f[:1]) + strings.ToLower(f[1:])
    }
    return strings.Join(fields, "")
}

func lowerCamel(s string) string {
    if s == "" { return s }
    return strings.ToLower(s[:1]) + s[1:]
}

func getStringAttr(span trace.Span, key string) string {
    if span.Attributes == nil { return "" }
    if v, ok := span.Attributes[key]; ok {
        if str, ok := v.(string); ok { return str }
    }
    return ""
}

