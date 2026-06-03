import { postFindings } from "$lib/api";
import {
  aggregateFindings,
  buildFindingsRunSummary,
  type FindingsFailure,
} from "$lib/findingsAggregator";
import { DEMO_WORKSPACE_PATH } from "$lib/projectStore";
import type {
  ChatMessage,
  ConnectorKind,
  ConnectorKnowledge,
  FindingsResult,
} from "$lib/types";
import { demoFindings } from "$lib/demoWorkspace";
import { assistantMsg, loadingMsg, progressMsg } from "$lib/chatController";

export type AnalysisSourceStatus = {
  connector: ConnectorKind;
  uri: string;
  status: "queued" | "running" | "done" | "failed";
  detail?: string;
};

export type AnalysisRunnerOptions = {
  workspacePath: string;
  readySources: ConnectorKnowledge[];
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

  if (options.readySources.length === 0) {
    options.openSources();
    options.addMessage(
      assistantMsg(
        "No ready sources in this workspace yet. Configure at least one source first.",
      ),
    );
    return;
  }

  const load = loadingMsg(
    `Running local analysis for ${options.readySources.length} selected source${options.readySources.length === 1 ? "" : "s"}...`,
  );
  options.addMessage(load);
  options.setBusy(true);
  try {
    const completed: FindingsResult[] = [];
    const failures: FindingsFailure[] = [];
    const sourceStatuses: AnalysisSourceStatus[] = options.readySources.map(
      (source) => ({
        connector: source.connector,
        uri: source.uri,
        status: "queued",
      }),
    );

    const updateProgress = () => {
      options.replaceMessage(
        load.id,
        progressMsg(load.id, buildAnalysisProgress(sourceStatuses)),
      );
    };
    updateProgress();

    for (const [index, source] of options.readySources.entries()) {
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
          const message = res.body.message ?? res.body.error ?? "unknown error";
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
      completedCount: completed.length,
      result: aggregated,
      failures,
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

export function buildAnalysisProgress(statuses: AnalysisSourceStatus[]) {
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
  return `Running local analysis... ${done}/${statuses.length} complete, ${failed} failed.\n${lines.join("\n")}`;
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

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
