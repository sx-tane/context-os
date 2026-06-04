package identity_test

import (
	"testing"

	"context-os/domain/types"
	"context-os/internal/stages/identity"
)

// TestCanonicalKeyNormalizesSeparators verifies equivalent spelling variants collapse to the same identity key.
func TestCanonicalKeyNormalizesSeparators(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "refund_status", want: "refundstatus"},
		{input: "Refund Status", want: "refundstatus"},
		{input: " refund-status ", want: "refundstatus"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := identity.CanonicalKey(tc.input)
			if got != tc.want {
				t.Errorf("CanonicalKey(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestResolveMergesAliasesInFirstSeenOrder verifies equivalent entities merge without losing alias order or confidence.
func TestResolveMergesAliasesInFirstSeenOrder(t *testing.T) {
	input := []types.Entity{
		{ID: "doc-1:refund_status", Name: "refund_status", SourceID: "doc-1"},
		{ID: "doc-1:refund-status", Name: "refund-status", SourceID: "doc-1"},
		{ID: "doc-1:paymentFlag", Name: "paymentFlag", SourceID: "doc-1"},
	}

	got := identity.Resolve(input)
	if len(got) != 2 {
		t.Fatalf("Resolve() length = %d, want 2", len(got))
	}
	if got[0].Entity.Name != "refund_status" {
		t.Fatalf("first canonical name = %q, want refund_status", got[0].Entity.Name)
	}
	if len(got[0].Entity.Aliases) != 2 {
		t.Fatalf("first aliases length = %d, want 2", len(got[0].Entity.Aliases))
	}
	if got[0].Entity.Aliases[0] != "refund_status" || got[0].Entity.Aliases[1] != "refund-status" {
		t.Fatalf("first aliases = %v, want [refund_status refund-status]", got[0].Entity.Aliases)
	}
	if got[0].Confidence != 1 || got[0].NeedsHuman {
		t.Fatalf("first resolution confidence = %v needs_human = %v, want 1 false", got[0].Confidence, got[0].NeedsHuman)
	}
}
