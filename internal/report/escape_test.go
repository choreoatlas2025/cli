// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package report

import "testing"

func TestXmlEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text with & ampersand", "text with &amp; ampersand"},
		{"<tag>content</tag>", "&lt;tag&gt;content&lt;/tag&gt;"},
		{`"quoted" and 'single'`, "&quot;quoted&quot; and &apos;single&apos;"},
		{"mixed <>&\"'", "mixed &lt;&gt;&amp;&quot;&apos;"},
	}

	for _, tt := range tests {
		result := xmlEscape(tt.input)
		if result != tt.expected {
			t.Errorf("xmlEscape(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
