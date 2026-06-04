import { postFindings } from "$lib/api";
import {
  aggregateFindings,
  buildFindingsRunSummary,
  type FindingsFailure,
} from "$lib/findings/aggregator";
import { DEMO_WORKSPACE_PATH } from "$lib/workspace/projectStore";
import type {
  Artifact,
  ChatMessage,
  ChatQueryResult,
  ConnectorKind,
  ConnectorKnowledge,
  FindingsResult,
} from "$lib/types";
import { demoFindings } from "$lib/chat/demoWorkspace";
import { assistantMsg, loadingMsg, progressMsg } from "$lib/chat/controller";
import {
  analysisSourceCountLabel,
  buildAnalysisSources,
} from "$lib/sources/analysisEligibility";

export type AnalysisSourceStatus = {
  connector: ConnectorKind;
  uri: string;
  status: "queued" | "running" | "done" | "failed";
  detail?: string;
};

export type AnalysisRunnerOptions = {
  workspacePath: string;
  readySources: ConnectorKnowledge[];
  lastChatResult?: ChatQueryResult | null;
  recentArtifacts?: Artifact[];
  addMessage: (message: ChatMessage) => void;
  replaceMessage: (id: string, message: ChatMessage) => void;
  setBusy: (busy: boolean) => void;
  setLastFindings: (result: FindingsResult | null) => void;
  setLastAnalysisAt: (value: string) => void;
  openSources: () => void;
  refreshWorkspace: () => Promise<void>;
  timeoutMs?: number;
};

const defaultAnalysisSourceTimeoutMs = 90_000;

export async function runAnalysis(options: AnalysisRunnerOptions) {
  if (options.workspacePath === DEMO_WORKSPACE_PATH) {
    const findings = demoFindings();
    options.setLastFindings(findings);
    options.setLastAnalysisAt(new Date().toISOString());
    options.addMessage(
      assistantMsg(
        "Demo analysis complete for 3 selected sources. Found 2 findings.",
        {
          kind: "findings",
          findingsResult: findings,
        },
      ),
    );
    return;
  }

  const { eligible, skipped } = buildAnalysisSources({
    readySources: options.readySources,
    lastChatResult: options.lastChatResult,
    recentArtifacts: options.recentArtifacts,
  });

  if (options.readySources.length === 0 && eligible.length === 0) {
    options.openSources();
    options.addMessage(
      assistantMsg(
        "No concrete sources are ready yet. Connect a source, or ask chat about a specific ticket, channel, repo, PR, document, folder, or file so saved evidence can be analyzed.",
      ),
    );
    return;
  }

  if (eligible.length === 0) {
    const summary = buildFindingsRunSummary({
      sourceCount: options.readySources.length,
      analysisSourceCount: 0,
      completedCount: 0,
      result: null,
      failures: [],
      skipped,
    });
    options.setLastFindings(null);
    options.setLastAnalysisAt(new Date().toISOString());
    options.addMessage(assistantMsg(summary));
    await options.refreshWorkspace();
    return;
  }

  const load = loadingMsg(
    `Running local analysis for ${analysisSourceCountLabel(eligible.length)}...`,
  );
  options.addMessage(load);
  options.setBusy(true);
  try {
    const completed: FindingsResult[] = [];
    const failures: FindingsFailure[] = [];
    const sourceStatuses: AnalysisSourceStatus[] = eligible.map(
      (source) => ({
        connector: source.connector,
        uri: source.uri,
        status: "queued",
      }),
    );

    const updateProgress = () => {
      options.replaceMessage(
        load.id,
        progressMsg(load.id, buildAnalysisProgress(sourceStatuses, skipped.length)),
      );
    };
    updateProgress();

    for (const [index, source] of eligible.entries()) {
      sourceStatuses[index] = {
        ...sourceStatuses[index],
        status: "running",
        detail: "request sent",
      };
      updateProgress();

      const controller = new AbortController();
      const timeout = window.setTimeout(
        () => controller.abort(),
        options.timeoutMs ?? defaultAnalysisSourceTimeoutMs,
      );
      try {
        const res = await postFindings(
          {
            workspace_id: options.workspacePath,
            connector: source.connector,
            uri: source.uri,
            provider: analysisProvider(source.connector),
            role: "pmo",
            include_execution: false,
          },
          { signal: controller.signal },
        );
        if (res.ok) {
          completed.push(res.body);
          sourceStatuses[index] = {
            ...sourceStatuses[index],
            status: "done",
            detail: `${res.body.event_count ?? 0} events, ${res.body.mismatch_count ?? res.body.mismatches?.length ?? 0} findings`,
          };
        } else {
          const message = findingsErrorMessage(res.body);
          failures.push({
            connector: source.connector,
            uri: source.uri,
            message,
          });
          sourceStatuses[index] = {
            ...sourceStatuses[index],
            status: "failed",
            detail: message,
          };
        }
      } catch (error) {
        const timeoutMs = options.timeoutMs ?? defaultAnalysisSourceTimeoutMs;
        const message = isAbortError(error)
          ? `timed out after ${Math.round(timeoutMs / 1000)}s`
          : String(error);
        failures.push({
          connector: source.connector,
          uri: source.uri,
          message,
        });
        sourceStatuses[index] = {
          ...sourceStatuses[index],
          status: "failed",
          detail: message,
        };
      } finally {
        window.clearTimeout(timeout);
        updateProgress();
      }
    }

    const aggregated = aggregateFindings(completed);
    options.setLastFindings(aggregated);
    options.setLastAnalysisAt(new Date().toISOString());
    const summary = buildFindingsRunSummary({
      sourceCount: options.readySources.length,
      analysisSourceCount: eligible.length,
      completedCount: completed.length,
      result: aggregated,
      failures,
      skipped,
    });

    options.replaceMessage(
      load.id,
      assistantMsg(summary, {
        kind: "findings",
        findingsResult: aggregated ?? undefined,
      }),
    );
  } catch (error) {
    options.replaceMessage(
      load.id,
      assistantMsg(`Analysis failed: ${String(error)}`),
    );
  } finally {
    options.setBusy(false);
    await options.refreshWorkspace();
  }
}

export function buildAnalysisProgress(
  statuses: AnalysisSourceStatus[],
  skippedCount = 0,
) {
  const done = statuses.filter((source) => source.status === "done").length;
  const failed = statuses.filter((source) => source.status === "failed").length;
  const lines = statuses.map((source, index) => {
    const label = `${index + 1}. ${source.connector}:${source.uri}`;
    if (source.status === "queued") return `${label} - queued`;
    if (source.status === "running") return `${label} - running`;
    if (source.status === "done") {
      return `${label} - done${source.detail ? ` (${source.detail})` : ""}`;
    }
    return `${label} - failed${source.detail ? `: ${source.detail}` : ""}`;
  });
  const skipText = skippedCount
    ? `, ${skippedCount} chat-only scope${skippedCount === 1 ? "" : "s"} skipped`
    : "";
  return `Running local analysis... ${done}/${statuses.length} complete, ${failed} failed${skipText}.\n${lines.join("\n")}`;
}

export function analysisProvider(connector: ConnectorKind) {
  const codexConnectors = new Set<ConnectorKind>([
    "github",
    "jira",
    "slack",
    "notion",
    "sharepoint",
    "googledrive",
  ]);
  return codexConnectors.has(connector) ? "codex" : "token";
}

function findingsErrorMessage(body: { error?: string; message?: string; examples?: string[] }) {
  if (body.error !== "source_too_broad") {
    return body.message ?? body.error ?? "unknown error";
  }
  const examples = body.examples?.filter(Boolean).slice(0, 3) ?? [];
  const suffix = examples.length ? ` Examples: ${examples.join(", ")}` : "";
  return `Choose a specific repo, project, issue, channel, thread, document, or folder.${suffix}`;
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
