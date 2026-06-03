import type {
  Artifact,
  ChatQueryResult,
  FindingsResult,
  GraphData,
  WorkspaceStatus,
} from "$lib/types";
import { DEMO_WORKSPACE_PATH } from "$lib/workspace/projectStore";

export function demoWorkspaceStatus(): WorkspaceStatus {
  return {
    workspace: {
      id: DEMO_WORKSPACE_PATH,
      name: "Demo Workspace",
      path: DEMO_WORKSPACE_PATH,
    },
    workspace_count: 2,
    event_count: 54,
    entity_count: 8,
    relationship_count: 7,
    mismatch_count: 2,
    connector_sync_count: 3,
    audit_count: 9,
    syncs: [
      {
        connector: "github",
        source_uri: "context-os/demo-api",
        event_count: 18,
        status: "ready",
      },
      {
        connector: "slack",
        source_uri: "#launch-review",
        event_count: 24,
        status: "ready",
      },
      {
        connector: "jira",
        source_uri: "DEMO",
        event_count: 12,
        status: "ready",
      },
    ],
  };
}

export function demoFindings(): FindingsResult {
  return {
    connector: "multiple",
    uri: "3 demo sources",
    role: "pmo",
    trace_id: "demo-trace",
    summary:
      "Demo findings show how ContextOS connects source evidence into cross-layer delivery risks.",
    event_count: 54,
    entity_count: 8,
    mismatch_count: 2,
    severity_count: { high: 1, medium: 1, low: 0 },
    mismatch_ids: ["demo-finding-1", "demo-finding-2"],
    mismatches: [
      {
        id: "demo-finding-1",
        severity: "high",
        mismatch_type: "requirement_gap",
        summary: "Checkout requirement missing service owner",
        description:
          "Jira says refund status must ship this sprint, but GitHub work only covers the UI state and Slack has an unresolved backend ownership question.",
        evidence: [
          "jira:DEMO-42",
          "github:context-os/demo-api#18",
          "slack:#launch-review",
        ],
        confidence: 0.88,
        impact:
          "PMO cannot confirm delivery readiness without backend ownership.",
        recommended_action:
          "Assign a service owner and update the Jira acceptance criteria before release review.",
      },
      {
        id: "demo-finding-2",
        severity: "medium",
        mismatch_type: "contract_drift",
        summary: "API contract drift for refundStatus",
        description:
          "Frontend discussion references refundStatus, while service notes still describe refund_state.",
        evidence: ["github:context-os/demo-api#21", "slack:#launch-review"],
        confidence: 0.76,
        impact: "QA may validate the wrong response field.",
        recommended_action:
          "Normalize the API contract name and add the field to the test plan.",
      },
    ],
  };
}

export function demoGraphData(): GraphData {
  return {
    workspace_id: DEMO_WORKSPACE_PATH,
    count: 8,
    entity_count: 8,
    relationship_count: 7,
    entities: [
      {
        id: "entity-checkout",
        name: "Checkout",
        type: "feature",
        source: "jira",
        confidence: 0.94,
        evidence: ["DEMO-42 acceptance criteria"],
      },
      {
        id: "entity-refund",
        name: "Refund Status",
        type: "requirement",
        source: "jira",
        confidence: 0.9,
        evidence: ["DEMO-42: show refund status"],
      },
      {
        id: "entity-api",
        name: "Payments API",
        type: "service",
        source: "github",
        confidence: 0.86,
        evidence: ["context-os/demo-api#18"],
      },
      {
        id: "entity-ui",
        name: "Checkout UI",
        type: "presentation",
        source: "github",
        confidence: 0.82,
        evidence: ["context-os/demo-api#21"],
      },
      {
        id: "entity-qa",
        name: "QA Release Plan",
        type: "qa",
        source: "jira",
        confidence: 0.78,
        evidence: ["DEMO-51"],
      },
      {
        id: "entity-pmo",
        name: "Launch Review",
        type: "pmo",
        source: "slack",
        confidence: 0.84,
        evidence: ["#launch-review decision thread"],
      },
      {
        id: "entity-owner",
        name: "Service Owner",
        type: "person",
        source: "slack",
        confidence: 0.63,
        evidence: ["Ownership unresolved in Slack"],
      },
      {
        id: "entity-contract",
        name: "refundStatus contract",
        type: "contract",
        source: "github",
        confidence: 0.72,
        evidence: ["OpenAPI notes in PR #21"],
      },
    ],
    relationships: [
      {
        id: "rel-1",
        from_id: "entity-checkout",
        to_id: "entity-refund",
        kind: "requires",
        confidence: 0.94,
        evidence: ["DEMO-42"],
      },
      {
        id: "rel-2",
        from_id: "entity-refund",
        to_id: "entity-api",
        kind: "depends_on",
        confidence: 0.86,
        evidence: ["PR #18"],
      },
      {
        id: "rel-3",
        from_id: "entity-ui",
        to_id: "entity-contract",
        kind: "expects_contract",
        confidence: 0.82,
        evidence: ["PR #21"],
      },
      {
        id: "rel-4",
        from_id: "entity-contract",
        to_id: "entity-api",
        kind: "implemented_by",
        confidence: 0.72,
        evidence: ["API notes"],
      },
      {
        id: "rel-5",
        from_id: "entity-qa",
        to_id: "entity-contract",
        kind: "validates",
        confidence: 0.78,
        evidence: ["DEMO-51"],
      },
      {
        id: "rel-6",
        from_id: "entity-pmo",
        to_id: "entity-checkout",
        kind: "tracks",
        confidence: 0.84,
        evidence: ["Launch review"],
      },
      {
        id: "rel-7",
        from_id: "entity-owner",
        to_id: "entity-api",
        kind: "owns",
        confidence: 0.42,
        evidence: ["Unconfirmed Slack thread"],
      },
    ],
  };
}

export function demoArtifacts(): Artifact[] {
  return [
    demoArtifact(
      "demo-artifact-1",
      "jira",
      "DEMO-42",
      "Refund status acceptance criteria",
      "PMO asks for refund status in checkout before launch review.",
      "2026-01-01T09:25:00.000Z",
    ),
    demoArtifact(
      "demo-artifact-2",
      "github",
      "context-os/demo-api#18",
      "Payments API ownership question",
      "Backend PR covers API plumbing but does not assign a service owner.",
      "2026-01-01T09:15:00.000Z",
    ),
    demoArtifact(
      "demo-artifact-3",
      "slack",
      "#launch-review",
      "Launch review decision thread",
      "Team agrees the UI is ready but backend ownership is still unresolved.",
      "2026-01-01T09:20:00.000Z",
    ),
    demoArtifact(
      "demo-artifact-4",
      "github",
      "context-os/demo-api#21",
      "refundStatus naming drift",
      "Frontend uses refundStatus while service notes mention refund_state.",
      "2026-01-01T09:10:00.000Z",
    ),
  ];
}

export function demoChatQueryResult(text: string): ChatQueryResult {
  const lower = text.toLowerCase();
  const artifacts = demoArtifacts();
  let answer =
    "Demo workspace is working locally. It has Jira, GitHub, and Slack evidence saved for the same workspace, so you can inspect findings, graph, and recent activity without connecting real sources.";
  let summary = "Demo workspace status";
  let intent = "status";

  if (
    lower.includes("finding") ||
    lower.includes("mismatch") ||
    lower.includes("refund")
  ) {
    intent = "findings";
    summary = "Demo refund status delivery risk";
    answer =
      "Jira says refund status must ship this sprint, GitHub currently covers the UI state, and Slack still has an unresolved backend ownership question. ContextOS flags that as a high-confidence requirement gap, with a second medium finding for refundStatus/refund_state contract drift.";
  } else if (
    lower.includes("graph") ||
    lower.includes("entity") ||
    lower.includes("relationship")
  ) {
    intent = "artifacts";
    summary = "Demo graph evidence";
    answer =
      "The demo graph links Checkout, Refund Status, Payments API, Checkout UI, QA Release Plan, Launch Review, Service Owner, and the refundStatus contract. The weakest link is service ownership, which is why the finding appears.";
  } else if (
    lower.includes("source") ||
    lower.includes("connected") ||
    lower.includes("ingest")
  ) {
    intent = "status";
    summary = "Demo source status";
    answer =
      "This demo workspace has 3 ready sources: Jira DEMO, GitHub context-os/demo-api, and Slack #launch-review. They are frontend demo records, so querying the demo does not call the backend workspace API.";
  }

  return {
    intent,
    workspace_id: DEMO_WORKSPACE_PATH,
    workspace_path: DEMO_WORKSPACE_PATH,
    provider: "local",
    answer,
    summary,
    artifact_count: artifacts.length,
    artifacts,
    syncs: demoWorkspaceStatus().syncs,
  };
}

function demoArtifact(
  id: string,
  connector: string,
  sourceURI: string,
  title: string,
  body: string,
  ingestedAt: string,
): Artifact {
  return {
    id,
    workspace_id: DEMO_WORKSPACE_PATH,
    connector,
    source_uri: sourceURI,
    event_type: "document.ingested",
    title,
    body,
    preview: body,
    content_hash: id,
    metadata: {},
    schema_version: "demo.v1",
    ingested_at: ingestedAt,
  };
}
