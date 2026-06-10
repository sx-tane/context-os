package chat

// White-box tests cover live scope selection helpers split out of chat.go.

import (
	"reflect"
	"testing"

	"context-os/domain/repository"
)

// TestConnectedLiveScopesOrdersAndFilters verifies fanout scopes are usable, non-filesystem, and stable ordered.
func TestConnectedLiveScopesOrdersAndFilters(t *testing.T) {
	t.Parallel()

	syncs := []repository.ConnectorSync{
		{Connector: "filesystem", SourceURI: "/tmp", Status: "idle"},
		{Connector: "slack", SourceURI: "slack://C1", Status: "error"},
		{Connector: "github", SourceURI: "owner/repo", Status: "idle"},
		{Connector: "jira", SourceURI: "jira://site/project:BKG", Status: "connected"},
	}

	got := connectedLiveScopes(syncs, nil)
	want := []repository.ConnectorSync{
		{Connector: "jira", SourceURI: "jira://site/project:BKG", Status: "connected"},
		{Connector: "github", SourceURI: "owner/repo", Status: "idle"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("connectedLiveScopes() = %#v, want %#v", got, want)
	}
}

// TestLiveFanoutScopesRespectsRequestedConnectors verifies requested connector filters constrain broad live fanout.
func TestLiveFanoutScopesRespectsRequestedConnectors(t *testing.T) {
	t.Parallel()

	syncs := []repository.ConnectorSync{
		{Connector: "github", SourceURI: "owner/repo", Status: "idle"},
		{Connector: "slack", SourceURI: "slack://C1", Status: "idle"},
	}
	got := liveFanoutScopes("check connected sources", []string{"slack"}, "", "", "", syncs)
	if len(got) != 1 {
		t.Fatalf("liveFanoutScopes() count = %d, want 1", len(got))
	}
	if got[0].Connector != "slack" {
		t.Fatalf("Connector = %q, want slack", got[0].Connector)
	}
}

// TestSourceMatchTokensIncludesSubtokens verifies source matching can resolve natural repo words.
func TestSourceMatchTokensIncludesSubtokens(t *testing.T) {
	t.Parallel()

	tokens := sourceMatchTokens("sx-tane/tourii-backend")
	for _, token := range []string{"sx-tane/tourii-backend", "tourii-backend", "tourii", "backend"} {
		if !tokens[token] {
			t.Fatalf("sourceMatchTokens() missing %q in %#v", token, tokens)
		}
	}
}
