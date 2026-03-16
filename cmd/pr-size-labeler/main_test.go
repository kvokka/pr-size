package main

import (
	"strings"
	"testing"
)

func TestPrivateKeyDiagnosticSummaryIsSafe(t *testing.T) {
	privateKey := "-----BEGIN RSA PRIVATE KEY-----\nsecret-line-1\nsecret-line-2\n-----END RSA PRIVATE KEY-----"
	summary := privateKeyDiagnosticSummary(privateKey)

	for _, want := range []string{`prefix="----"`, "begin_marker=true", "end_marker=true", "newline_count=3", "contains_escaped_newline=false", "contains_carriage_return=false"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q: %s", want, summary)
		}
	}
	for _, forbidden := range []string{"secret-line-1", "secret-line-2", "RSA PRIVATE KEY-----\nsecret"} {
		if strings.Contains(summary, forbidden) {
			t.Fatalf("summary leaked private key content %q: %s", forbidden, summary)
		}
	}
}
