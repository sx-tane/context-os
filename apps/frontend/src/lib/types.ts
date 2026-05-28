export type ServiceStatus = "checking" | "ok" | "unreachable";

export type IngestProvider = "token" | "codex";

export interface CodexPlugin {
  name: string;
  installed: boolean;
  enabled: boolean;
}

export interface IngestEvent {
  id: string;
  type: string;
  source: string;
  source_id: string;
  subject: string;
  occurred_at: string;
}

export interface IngestResult {
  connector: string;
  capabilities: string[];
  event: IngestEvent;
  preview: string;
  metadata: Record<string, string>;
}
