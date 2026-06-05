import type {
  EvidenceBasketItem,
  FindingActionItem,
} from "$lib/workflow/types";

export interface AnalysisBasketPayload {
  workspace_id: string;
  items: EvidenceBasketItem[];
}

export interface FindingActionsPayload {
  workspace_id: string;
  actions: FindingActionItem[];
}

export interface ChatQueryRequest {
  workspace_id: string;
  workspace_path?: string;
  message: string;
  connector?: string;
  connectors?: string[];
  source_uri?: string;
  mode?: "auto" | "codex" | "local";
  timezone?: string;
  local_date?: string;
  response_language?: string;
  limit?: number;
}

export interface ChatSessionResetRequest {
  workspace_id: string;
  workspace_path?: string;
}
