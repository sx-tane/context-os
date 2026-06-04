import type {
  ConnectorKnowledge,
  KnowledgeStatus,
  WorkspaceSyncState,
} from "$lib/types";

export interface ConnectorSyncMappingResult {
  connectors: ConnectorKnowledge[];
  changed: boolean;
}

export function applyWorkspaceSyncsToConnectors(
  connectors: ConnectorKnowledge[],
  syncs: WorkspaceSyncState[] | undefined,
): ConnectorSyncMappingResult {
  let changed = false;
  const updated = connectors.map((knowledge) => {
    const sync = syncs?.find((item) => connectorSyncMatchesKnowledge(item, knowledge));
    if (sync) {
      const next = connectorKnowledgeFromSync(knowledge, sync);
      changed = changed || connectorKnowledgeChanged(knowledge, next);
      return next;
    }
    if (knowledge.status === "ready") {
      changed = true;
      return {
        ...knowledge,
        status: "configuring" as KnowledgeStatus,
        eventCount: 0,
        error: "Not confirmed in the workspace database.",
      };
    }
    return knowledge;
  });
  return { connectors: updated, changed };
}

function connectorKnowledgeFromSync(
  knowledge: ConnectorKnowledge,
  sync: WorkspaceSyncState,
): ConnectorKnowledge {
  if (sync.status === "error") {
    return {
      ...knowledge,
      status: "error",
      eventCount: sync.event_count ?? knowledge.eventCount,
      error: sync.last_error || "Workspace database reports this source has an error.",
    };
  }
  return {
    ...knowledge,
    status: "ready",
    eventCount: eventCountForSavedSource(sync, knowledge.eventCount),
    error: undefined,
  };
}

function connectorSyncMatchesKnowledge(
  sync: WorkspaceSyncState,
  knowledge: ConnectorKnowledge,
): boolean {
  if (sync.connector !== knowledge.connector) return false;
  if (sync.source_uri === knowledge.uri || sync.source_uri === "" || !sync.source_uri) {
    return true;
  }
  if (knowledge.connector !== "filesystem") return false;
  const syncURI = sync.source_uri.trim();
  const knowledgeURI = knowledge.uri.trim();
  if (!syncURI || !knowledgeURI) return false;
  return syncURI.includes(knowledgeURI) ||
    knowledgeURI.includes(syncURI) ||
    (sync.event_count ?? 0) > 0;
}

function eventCountForSavedSource(sync: WorkspaceSyncState, fallback?: number): number | undefined {
  if (sync.status === "connected" || sync.status === "pending") {
    return (sync.event_count ?? 0) > 0 ? sync.event_count : undefined;
  }
  return sync.event_count ?? fallback;
}

function connectorKnowledgeChanged(
  current: ConnectorKnowledge,
  next: ConnectorKnowledge,
): boolean {
  return next.status !== current.status ||
    next.eventCount !== current.eventCount ||
    next.error !== current.error;
}
