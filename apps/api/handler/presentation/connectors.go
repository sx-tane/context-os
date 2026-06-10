package presentation

import (
	"context-os/apps/api/request"
	"context-os/domain/contracts"
	codexsource "context-os/internal/source/codex"
	filesystemsource "context-os/internal/source/filesystem"
	githubsource "context-os/internal/source/github"
	googledrivesource "context-os/internal/source/googledrive"
	jirasource "context-os/internal/source/jira"
	notionsource "context-os/internal/source/notion"
	sharepointsource "context-os/internal/source/sharepoint"
	slacksource "context-os/internal/source/slack"
	"fmt"
	"strings"
)

// resolveConnector builds the requested source connector and attaches provider-specific metadata.
func resolveConnector(req request.PresentationFindings, metadata map[string]string) (contracts.MCPSourceConnector, error) {
	connector := strings.ToLower(strings.TrimSpace(req.Connector))
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = "token"
	}

	token := strings.TrimSpace(req.Token)
	if provider == "codex" {
		if token != "" {
			metadata[codexsource.MetadataTokenOverride] = token
		}
		switch connector {
		case "github":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginGitHub
		case "jira":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginAtlassianRovo
		case "slack":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginSlack
		case "googledrive":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginGoogleDrive
		case "notion":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginNotion
		case "sharepoint":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginSharePoint
		default:
			return nil, fmt.Errorf("connector %q does not support provider=codex", connector)
		}
		return codexsource.NewConnector(), nil
	}

	switch connector {
	case "github":
		setIfNotEmpty(metadata, "github_token", token)
		return githubsource.NewConnector(), nil
	case "jira":
		setIfNotEmpty(metadata, "jira_token", token)
		return jirasource.NewConnector(), nil
	case "slack":
		setIfNotEmpty(metadata, "slack_token", token)
		return slacksource.NewConnector(), nil
	case "filesystem":
		return filesystemsource.NewConnector(), nil
	case "googledrive":
		setIfNotEmpty(metadata, googledrivesource.MetadataAccessToken, token)
		return googledrivesource.NewConnector(), nil
	case "notion":
		setIfNotEmpty(metadata, notionsource.MetadataToken, token)
		return notionsource.NewConnector(), nil
	case "sharepoint":
		setIfNotEmpty(metadata, sharepointsource.MetadataAccessToken, token)
		return sharepointsource.NewConnector(), nil
	default:
		return nil, fmt.Errorf("unsupported connector %q", connector)
	}
}

// broadCodexSource rejects connector-only Codex requests that lack a concrete source scope.
func broadCodexSource(req request.PresentationFindings) (bool, []string) {
	if !strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		return false, nil
	}
	connector := strings.ToLower(strings.TrimSpace(req.Connector))
	uri := strings.ToLower(strings.TrimSpace(req.URI))
	if uri == "" || uri != connector {
		return false, nil
	}
	switch connector {
	case "github":
		return true, []string{"https://github.com/owner/repo", "https://github.com/owner/repo/pull/123", "https://github.com/owner/repo/issues/123"}
	case "jira":
		return true, []string{"BKGDEV-8466", "https://example.atlassian.net/browse/BKGDEV-8466", "project:BKGDEV"}
	case "slack":
		return true, []string{"slack://C12345678", "slack://C12345678/p1717000000000000"}
	case "googledrive":
		return true, []string{"https://drive.google.com/file/d/FILE_ID/view", "https://drive.google.com/drive/folders/FOLDER_ID"}
	case "notion":
		return true, []string{"https://www.notion.so/workspace/Page-0123456789abcdef0123456789abcdef", "notion://page/0123456789abcdef0123456789abcdef"}
	case "sharepoint":
		return true, []string{"sharepoint://sites/site-id/items/item-id", "https://tenant.sharepoint.com/sites/team/Shared%20Documents/spec.docx"}
	default:
		return false, nil
	}
}
