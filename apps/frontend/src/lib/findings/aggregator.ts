import type { ConnectorKind, FindingsResult } from "$lib/types";

export interface FindingsFailure {
  connector: ConnectorKind;
  uri: string;
  message: string;
}

export interface FindingsSkipped {
  connector: ConnectorKind;
  uri: string;
  reason: string;
}

export interface FindingsRunSummary {
  result: FindingsResult | null;
  message: string;
}

export function aggregateFindings(results: FindingsResult[]): FindingsResult | null {
  if (results.length === 0) return null;

  const mismatches = results.flatMap((result) => result.mismatches ?? []);
  const mismatchCount = results.reduce(
    (sum, result) => sum + (result.mismatch_count ?? result.mismatches?.length ?? 0),
    0,
  );
  const eventCount = results.reduce((sum, result) => sum + (result.event_count ?? 0), 0);
  const entityCount = results.reduce((sum, result) => sum + (result.entity_count ?? 0), 0);
  const severityCount = {
    high: results.reduce((sum, result) => sum + (result.severity_count?.high ?? 0), 0),
    medium: results.reduce((sum, result) => sum + (result.severity_count?.medium ?? 0), 0),
    low: results.reduce((sum, result) => sum + (result.severity_count?.low ?? 0), 0),
  };
  const mismatchIDs = [
    ...new Set(results.flatMap((result) => result.mismatch_ids ?? [])),
  ];

  return {
    connector: results.length === 1 ? results[0].connector : "multiple",
    uri: results.length === 1 ? results[0].uri : `${results.length} sources`,
    role: results[0].role,
    trace_id: results.map((result) => result.trace_id).filter(Boolean).join(","),
    summary:
      mismatchCount > 0
        ? `Aggregated ${mismatchCount} mismatch signal${mismatchCount === 1 ? "" : "s"} across ${results.length} source${results.length === 1 ? "" : "s"}.`
        : `Analysis ran, no mismatch signals detected across ${results.length} source${results.length === 1 ? "" : "s"}.`,
    mismatches,
    event_count: eventCount,
    entity_count: entityCount,
    mismatch_count: mismatchCount,
    severity_count: severityCount,
    mismatch_ids: mismatchIDs,
  };
}

export function buildFindingsRunSummary(params: {
  sourceCount: number;
  analysisSourceCount?: number;
  completedCount: number;
  result: FindingsResult | null;
  failures: FindingsFailure[];
  skipped?: FindingsSkipped[];
}): string {
  const mismatchCount = params.result?.mismatch_count ?? 0;
  const eventCount = params.result?.event_count ?? 0;
  const entityCount = params.result?.entity_count ?? 0;
  const analysisSourceCount = params.analysisSourceCount ?? params.sourceCount;
  const sourceWord = analysisSourceCount === 1 ? "source" : "sources";
  const findingWord = mismatchCount === 1 ? "finding" : "findings";
  let base = "";
  if (analysisSourceCount === 0) {
    base = `Analysis skipped: 0 concrete sources were ready. Ask chat about a specific ticket, channel, repo, PR, document, folder, or file so saved evidence can become analysis-ready.`;
  } else if (mismatchCount > 0) {
    base = `Analysis complete for ${params.completedCount}/${analysisSourceCount} concrete ${sourceWord}. Found ${mismatchCount} ${findingWord}.`;
  } else {
    base = `Analysis ran, no mismatch signals detected across ${params.completedCount}/${analysisSourceCount} concrete ${sourceWord}. Sources: ${params.completedCount}. Events: ${eventCount}. Entities: ${entityCount}.`;
  }

  const sections = [base];
  if (params.skipped?.length) {
    sections.push(
      `Skipped chat-only scopes:\n- ${params.skipped
        .map((source) => `${source.connector}:${source.uri} - ${source.reason}`)
        .join("\n- ")}`,
    );
  }
  if (params.failures.length) {
    sections.push(
      `Failed:\n- ${params.failures
        .map((failure) => `${failure.connector}:${failure.uri} - ${failure.message}`)
        .join("\n- ")}`,
    );
  }
  return sections.join("\n\n");
}
