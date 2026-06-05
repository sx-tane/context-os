import type { ConnectorKind } from "$lib/types";

export type EvidenceOrigin = "chat" | "activity" | "source" | "manual";

export interface EvidenceBasketItem {
  id: string;
  connector: ConnectorKind;
  uri: string;
  label: string;
  origin: EvidenceOrigin | string;
  artifactId?: string;
  messageId?: string;
  addedAt: string;
}

export type FindingActionStatus =
  | "open"
  | "checking"
  | "done"
  | "ignored"
  | "false_positive";

export interface FindingActionItem {
  findingId: string;
  status: FindingActionStatus;
  note?: string;
  updatedAt: string;
}
