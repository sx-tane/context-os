import type {
  AnswerSection,
  Artifact,
  ChatQueryResult,
  EvidenceBasketItem,
  FindingActionItem,
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
      "demo-artifact-0",
      "demo",
      "contextos-demo://planning-notes",
      "Planning mode and agent chat tour",
      "Demo notes describe the planning-first workflow, source cards, Activity cleanup, stream transcript behavior, findings, graph, and agent chat mode.",
      "2026-01-01T09:10:00.000Z",
      "tour_note",
    ),
    demoArtifact(
      "demo-artifact-1",
      "jira",
      "DEMO-42",
      "Refund status acceptance criteria",
      "PMO asks for refund status in checkout before launch review.",
      "2026-01-01T09:25:00.000Z",
      "requirement",
    ),
    demoArtifact(
      "demo-artifact-2",
      "github",
      "context-os/demo-api#18",
      "Payments API ownership question",
      "Backend PR covers API plumbing but does not assign a service owner.",
      "2026-01-01T09:15:00.000Z",
      "pull_request",
    ),
    demoArtifact(
      "demo-artifact-3",
      "slack",
      "#launch-review",
      "Launch review decision thread",
      "Team agrees the UI is ready but backend ownership is still unresolved.",
      "2026-01-01T09:20:00.000Z",
      "thread",
    ),
    demoArtifact(
      "demo-artifact-4",
      "github",
      "context-os/demo-api#21",
      "refundStatus naming drift",
      "Frontend uses refundStatus while service notes mention refund_state.",
      "2026-01-01T09:10:00.000Z",
      "pull_request",
    ),
    demoArtifact(
      "demo-artifact-5",
      "demo",
      "contextos-demo://activity-cleanup",
      "Activity cleanup example",
      "The demo keeps Activity read-only but shows how live evidence cleanup is exposed for old noisy local records.",
      "2026-01-01T09:29:00.000Z",
      "workflow_note",
    ),
  ];
}

export function demoAnalysisBasket(): EvidenceBasketItem[] {
  return [
    {
      id: "jira:DEMO-42",
      connector: "jira",
      uri: "DEMO-42",
      label: "Refund status acceptance criteria",
      origin: "activity",
      artifactId: "demo-artifact-1",
      addedAt: "2026-01-01T09:32:00.000Z",
    },
    {
      id: "github:context-os/demo-api#18",
      connector: "github",
      uri: "context-os/demo-api#18",
      label: "Payments API ownership question",
      origin: "activity",
      artifactId: "demo-artifact-2",
      addedAt: "2026-01-01T09:33:00.000Z",
    },
    {
      id: "slack:#launch-review",
      connector: "slack",
      uri: "#launch-review",
      label: "Launch review decision thread",
      origin: "activity",
      artifactId: "demo-artifact-3",
      addedAt: "2026-01-01T09:34:00.000Z",
    },
  ];
}

export function demoFindingActions(): FindingActionItem[] {
  return [
    {
      findingId: "demo-finding-1",
      status: "checking",
      updatedAt: "2026-01-01T09:35:00.000Z",
    },
    {
      findingId: "demo-finding-2",
      status: "open",
      updatedAt: "2026-01-01T09:35:00.000Z",
    },
  ];
}

export function demoPlanningTourResult(): ChatQueryResult {
  const artifacts = demoArtifacts();
  const answer =
    "Demo planning mode starts here. Use the agent chat to inspect source cards, evidence basket, analysis preview, finding checklist, Markdown export, graph, Activity filters, stream progress, and cleanup behavior without connecting live sources.";
  return {
    intent: "demo_tour",
    workspace_id: DEMO_WORKSPACE_PATH,
    workspace_path: DEMO_WORKSPACE_PATH,
    provider: "local",
    answer,
    summary: "Demo planning and function tour",
    answer_sections: [
      {
        source_label: "Demo · Planning notes",
        connector: "demo",
        source_uri: "contextos-demo://planning-notes",
        summary:
          "The demo opens with planning mode first so the available functions are visible before live setup.",
        facts: [
          "Agent chat can answer against demo Jira, GitHub, Slack, graph, findings, and Activity data.",
          "The demo opens with three evidence items already pinned so Analysis Preview shows basket-only analysis.",
          "Findings include checklist status so open/checking/done behavior is visible immediately.",
          "Structured source cards are rendered from answer_sections instead of parsing prose.",
          "The demo is local-only and does not require API, worker, or Codex availability.",
        ],
        open_items: [
          "Real backend agent planning mode is not implemented in this demo pass.",
        ],
        coding_notes: [
          "Ask for planning mode, agent mode, functions, source cards, stream, cleanup, findings, graph, or sources.",
          "Try asking for basket, preview, export, checklist, or Activity filters.",
        ],
        links: ["contextos-demo://planning-notes"],
        timestamps: ["2026-01-01T09:10:00.000Z"],
        confidence: 1,
        status: "demo",
      },
    ],
    artifact_count: artifacts.length,
    artifacts,
    syncs: demoWorkspaceStatus().syncs,
  };
}

export function demoChatQueryResult(text: string): ChatQueryResult {
  const lower = text.toLowerCase();
  const artifacts = demoArtifacts();
  let answer =
    "Demo workspace is working locally. It has Jira, GitHub, and Slack evidence saved for the same workspace, so you can inspect findings, graph, and recent activity without connecting real sources.";
  let summary = "Demo workspace status";
  let intent = "status";
  let answer_sections: AnswerSection[] = demoStatusSections();

  if (
    lower.includes("planning") ||
    lower.includes("plan mode") ||
    lower.includes("agent") ||
    lower.includes("function") ||
    lower.includes("note") ||
    lower.includes("walkthrough")
  ) {
    return demoPlanningTourResult();
  } else if (
    lower.includes("basket") ||
    lower.includes("bucket") ||
    lower.includes("pinned") ||
    lower.includes("pin") ||
    lower.includes("preview") ||
    lower.includes("export") ||
    lower.includes("checklist") ||
    lower.includes("workflow")
  ) {
    intent = "status";
    summary = "Demo workflow showcase";
    answer =
      "The demo has three pinned evidence sources in the Evidence Basket. Analysis Preview shows those as Included, keeps other concrete sources visible as available, and lets you export a Markdown snapshot. Findings also show action checklist status.";
    answer_sections = demoWorkflowSections();
  } else if (
    lower.includes("finding") ||
    lower.includes("mismatch") ||
    lower.includes("refund")
  ) {
    intent = "findings";
    summary = "Demo refund status delivery risk";
    answer =
      "Jira says refund status must ship this sprint, GitHub currently covers the UI state, and Slack still has an unresolved backend ownership question. ContextOS flags that as a high-confidence requirement gap, with a second medium finding for refundStatus/refund_state contract drift.";
    answer_sections = demoFindingSections();
  } else if (
    lower.includes("graph") ||
    lower.includes("entity") ||
    lower.includes("relationship")
  ) {
    intent = "artifacts";
    summary = "Demo graph evidence";
    answer =
      "The demo graph links Checkout, Refund Status, Payments API, Checkout UI, QA Release Plan, Launch Review, Service Owner, and the refundStatus contract. The weakest link is service ownership, which is why the finding appears.";
    answer_sections = demoGraphSections();
  } else if (
    lower.includes("source") ||
    lower.includes("card") ||
    lower.includes("connected") ||
    lower.includes("ingest")
  ) {
    intent = "status";
    summary = "Demo source status";
    answer =
      "This demo workspace has 3 ready sources: Jira DEMO, GitHub context-os/demo-api, and Slack #launch-review. They are frontend demo records, so querying the demo does not call the backend workspace API.";
    answer_sections = demoStatusSections();
  } else if (
    lower.includes("activity") ||
    lower.includes("cleanup") ||
    lower.includes("clean noisy")
  ) {
    intent = "artifacts";
    summary = "Demo Activity and cleanup behavior";
    answer =
      "Demo Activity is read-only and shows clean source labels for the seeded notes, Jira issue, GitHub PRs, and Slack thread. The cleanup action is visible for live workspaces and removes old noisy live-evidence rows only after confirmation.";
    answer_sections = demoActivitySections();
  } else if (lower.includes("stream") || lower.includes("progress")) {
    intent = "status";
    summary = "Demo stream behavior";
    answer =
      "Live workspaces show compact stream progress while Codex runs and cap expanded transcript rendering for readability. The demo explains the behavior without starting a backend stream.";
    answer_sections = demoStreamSections();
  }

  return {
    intent,
    workspace_id: DEMO_WORKSPACE_PATH,
    workspace_path: DEMO_WORKSPACE_PATH,
    provider: "local",
    answer,
    summary,
    answer_sections,
    artifact_count: artifacts.length,
    artifacts,
    syncs: demoWorkspaceStatus().syncs,
  };
}

function demoStatusSections(): AnswerSection[] {
  return [
    {
      source_label: "Jira · DEMO",
      connector: "jira",
      source_uri: "DEMO",
      summary:
        "Seeded Jira evidence provides the refund status requirement and QA release plan.",
      facts: [
        "DEMO-42 asks for refund status in checkout before launch review.",
        "DEMO-51 tracks QA validation for the API response field.",
      ],
      open_items: ["Backend ownership is still unresolved."],
      links: ["DEMO-42", "DEMO-51"],
      timestamps: ["2026-01-01T09:25:00.000Z"],
      confidence: 0.9,
      status: "ready",
    },
    {
      source_label: "GitHub · context-os/demo-api",
      connector: "github",
      source_uri: "context-os/demo-api",
      summary:
        "Seeded GitHub evidence shows UI implementation and API contract naming discussion.",
      facts: [
        "PR #18 covers the refund status UI state.",
        "PR #21 references refundStatus while service notes still mention refund_state.",
      ],
      coding_notes: [
        "Normalize the API field name before QA locks the release plan.",
      ],
      links: ["context-os/demo-api#18", "context-os/demo-api#21"],
      timestamps: ["2026-01-01T09:15:00.000Z"],
      confidence: 0.84,
      status: "ready",
    },
    {
      source_label: "Slack · #launch-review",
      connector: "slack",
      source_uri: "#launch-review",
      summary:
        "Seeded Slack evidence captures the unresolved service ownership question.",
      facts: ["Launch review is blocked until the Payments API owner is confirmed."],
      open_items: ["Assign the owner and update Jira acceptance criteria."],
      links: ["#launch-review"],
      timestamps: ["2026-01-01T09:20:00.000Z"],
      confidence: 0.82,
      status: "ready",
    },
  ];
}

function demoFindingSections(): AnswerSection[] {
  return [
    {
      source_label: "Finding · Requirement gap",
      connector: "multiple",
      source_uri: "demo-finding-1",
      summary:
        "Refund status is required this sprint, but backend service ownership is still missing.",
      facts: [
        "Jira DEMO-42 defines the PMO requirement.",
        "GitHub PR #18 covers UI state only.",
        "Slack #launch-review leaves backend ownership unresolved.",
      ],
      open_items: [
        "Assign a Payments API owner before release review.",
        "Update Jira acceptance criteria after ownership is confirmed.",
      ],
      links: ["DEMO-42", "context-os/demo-api#18", "#launch-review"],
      confidence: 0.88,
      status: "high",
    },
    {
      source_label: "Finding · Contract drift",
      connector: "multiple",
      source_uri: "demo-finding-2",
      summary:
        "Frontend and service notes disagree on refundStatus versus refund_state.",
      facts: [
        "Frontend discussion references refundStatus.",
        "Service notes still describe refund_state.",
      ],
      coding_notes: [
        "Choose one response field and add it to the QA release plan.",
      ],
      links: ["context-os/demo-api#21", "#launch-review"],
      confidence: 0.76,
      status: "medium",
    },
  ];
}

function demoGraphSections(): AnswerSection[] {
  return [
    {
      source_label: "Graph · Refund Status",
      connector: "graph",
      source_uri: "contextos-demo://graph/refund-status",
      summary:
        "The graph connects the refund requirement to UI, API, QA, PMO, and owner entities.",
      facts: [
        "8 entities and 7 relationships are seeded.",
        "The weakest relationship is Service Owner owns Payments API.",
      ],
      open_items: ["Confirm the service owner to strengthen the graph signal."],
      confidence: 0.84,
      status: "demo",
    },
  ];
}

function demoActivitySections(): AnswerSection[] {
  return [
    {
      source_label: "Activity · Clean source records",
      connector: "demo",
      source_uri: "contextos-demo://activity",
      summary:
        "Activity uses one readable source record per seeded note, issue, PR, or thread.",
      facts: [
        "Demo Activity is read-only.",
        "Connector, source URI, evidence type, and keyword filters can be tried against seeded demo rows.",
        "Live workspaces expose a confirmation-gated cleanup action.",
      ],
      open_items: [
        "Cleanup never runs automatically; the user must choose it from Activity.",
      ],
      links: ["contextos-demo://activity-cleanup"],
      confidence: 1,
      status: "demo",
    },
  ];
}

function demoWorkflowSections(): AnswerSection[] {
  return [
    {
      source_label: "Workflow · Evidence Basket",
      connector: "jira",
      source_uri: "DEMO-42",
      summary:
        "The basket pins the concrete sources that should drive the next analysis run.",
      facts: [
        "Pinned demo sources are Jira DEMO-42, GitHub PR #18, and Slack #launch-review.",
        "When basket has items, Run Analysis uses basket sources only.",
        "Preview still shows other concrete sources as available but not selected.",
      ],
      open_items: [
        "Remove a basket item from Preview to see the included source list change.",
      ],
      links: ["DEMO-42", "context-os/demo-api#18", "#launch-review"],
      confidence: 1,
      status: "demo",
    },
    {
      source_label: "Workflow · Checklist and Export",
      connector: "github",
      source_uri: "context-os/demo-api#21",
      summary:
        "Findings keep action status beside the recommendation, and Export Markdown creates a handoff snapshot from loaded state.",
      facts: [
        "Finding 1 starts as checking; finding 2 starts as open.",
        "The Copy action produces a share-ready finding summary.",
        "Export Markdown includes source health, basket, findings, checklist, graph counts, and recent Activity.",
      ],
      links: ["context-os/demo-api#21"],
      confidence: 1,
      status: "demo",
    },
  ];
}

function demoStreamSections(): AnswerSection[] {
  return [
    {
      source_label: "Agent stream · Live workspace behavior",
      connector: "codex",
      source_uri: "contextos-demo://stream",
      summary:
        "Live Codex queries show compact progress and keep expanded stream output bounded.",
      facts: [
        "The latest stream line remains visible while collapsed.",
        "Expanded stream rendering is capped to avoid Show/Hide lag.",
      ],
      coding_notes: [
        "Demo mode does not call /chat/query/stream; it describes the behavior locally.",
      ],
      confidence: 1,
      status: "demo",
    },
  ];
}

function demoArtifact(
  id: string,
  connector: string,
  sourceURI: string,
  title: string,
  body: string,
  ingestedAt: string,
  evidenceKind = "document",
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
    metadata: { evidence_kind: evidenceKind },
    schema_version: "demo.v1",
    ingested_at: ingestedAt,
  };
}
