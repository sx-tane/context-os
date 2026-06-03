<script lang="ts">
    import { onMount } from "svelte";
    import type {
        Artifact,
        ChatMessage,
        ChatQueryResult,
        CodexPlugin,
        ConnectorKnowledge,
        ConnectorKind,
        FindingsResult,
        GraphData,
        GraphEntity,
        GraphRelationship,
        ServiceStatus,
        WorkspaceStatus,
    } from "$lib/types";
    import {
        API_URL,
        deleteWorkspace,
        getArtifacts,
        getGraphData,
        getWorkspaceStatus,
        postChatQuery,
        postFindings,
        probeService,
    } from "$lib/api";
    import {
        addMessage,
        chatMessages,
        clearChat,
        DEFAULT_WORKSPACE_PATH,
        DEMO_WORKSPACE_PATH,
        addWorkspace,
        getProject,
        hydrateWorkspaces,
        loadWorkspaceStatus,
        openProject,
        project,
        replaceMessage,
        removeWorkspace,
        workspaces,
    } from "$lib/projectStore";
    import {
        aggregateFindings,
        buildFindingsRunSummary,
        type FindingsFailure,
    } from "$lib/findingsAggregator";
    import KnowledgeInstall from "$lib/components/knowledge/KnowledgeInstall.svelte";

    type GraphLink = {
        id: string;
        source: string;
        target: string;
        label: string;
        strength: number;
        evidence?: string[];
    };
    type GraphTypeSummary = {
        type: string;
        count: number;
        color: string;
    };
    type EntityIndexItem = GraphEntity & {
        degree: number;
    };
    type EntityIndexSection = {
        label: string;
        entities: EntityIndexItem[];
    };
    type RelationshipRow = {
        entityName: string;
        confidence: number;
        evidence?: string[];
    };
    type RelationshipKindGroup = {
        kind: string;
        incoming: RelationshipRow[];
        outgoing: RelationshipRow[];
    };
    type FocusGraphRow = {
        id: string;
        side: "incoming" | "outgoing";
        y: number;
        entity: GraphEntity;
        link: GraphLink;
        color: string;
    };
    type AnalysisSourceStatus = {
        connector: ConnectorKind;
        uri: string;
        status: "queued" | "running" | "done" | "failed";
        detail?: string;
    };

    const analysisSourceTimeoutMs = 90_000;

    let apiStatus: ServiceStatus = "checking";
    let workerStatus: ServiceStatus = "checking";
    let codexLoggedIn = false;
    let codexInstalled = false;
    let codexAccount = "";
    let codexPlugins: CodexPlugin[] = [];
    let workspacePath = DEFAULT_WORKSPACE_PATH;
    let newWorkspacePath = "";
    let command = "";
    let busy = false;
    let sourcePanelOpen = false;
    let activeInsightTab: "findings" | "graph" | "activity" = "findings";
    let workspaceStatus: WorkspaceStatus | null = null;
    let graphData: GraphData | null = null;
    let selectedEntity: GraphEntity | null = null;
    let entityQuery = "";
    let lastChatResult: ChatQueryResult | null = null;
    let lastFindings: FindingsResult | null = null;
    let lastAnalysisAt = "";
    let activityArtifacts: Artifact[] = [];
    let removeConfirmOpen = false;
    let workspacePendingRemoval = "";
    let removeInProgress = false;
    let walkthroughOpen = false;

    $: readySources = $project.connectors.filter(
        (source) => source.status === "ready",
    );
    $: graphEntities = graphData?.entities ?? [];
    $: graphRelationships = graphData?.relationships ?? [];
    $: graphLinks = buildGraphLinks(graphEntities, graphRelationships);
    $: graphEntityById = new Map(graphEntities.map((entity) => [entity.id, entity]));
    $: graphDegree = linkDegree(graphLinks);
    $: selectedLinks = selectedEntity
        ? graphLinks.filter((link) => link.source === selectedEntity?.id || link.target === selectedEntity?.id)
        : [];
    $: linkedEntityIds = selectedEntity ? linkedIdsForEntity(selectedEntity.id, selectedLinks) : new Set<string>();
    $: entityIndexSections = buildEntityIndexSections(
        graphEntities,
        graphDegree,
        selectedEntity,
        linkedEntityIds,
        entityQuery,
    );
    $: relationshipGroups = selectedEntity
        ? buildRelationshipGroups(selectedEntity, selectedLinks, graphEntityById)
        : [];
    $: focusRows = selectedEntity
        ? buildFocusGraphRows(selectedEntity, selectedLinks, graphEntityById)
        : [];
    $: incomingFocusRows = focusRows.filter((row) => row.side === "incoming");
    $: outgoingFocusRows = focusRows.filter((row) => row.side === "outgoing");
    $: graphLegendTypes = buildGraphLegendTypes(graphEntities);
    $: if (
        graphEntities.length > 0 &&
        (!selectedEntity || !graphEntities.some((entity) => entity.id === selectedEntity?.id))
    ) {
        selectedEntity = topGraphEntity(graphEntities, graphDegree);
    }
    $: if (graphEntities.length === 0) {
        selectedEntity = null;
    }
    $: mismatchCount =
        workspaceStatus?.mismatch_count ?? lastFindings?.mismatch_count ?? 0;
    $: statusLine = buildStatusLine(
        apiStatus,
        workerStatus,
        codexInstalled,
        codexLoggedIn,
        codexAccount,
    );
    $: codexLabel = codexLoggedIn ? normalizeCodexAccount(codexAccount) : "Codex not logged in";
    $: sourceSummary = `${readySources.length} source${readySources.length === 1 ? "" : "s"}`;
    $: topContext = `${$project.name} · ${sourceSummary}`;
    $: hasSources = readySources.length > 0;
    $: sourceGroups = groupSources(readySources);
    $: recentArtifacts = activityArtifacts.length > 0 ? activityArtifacts : (lastChatResult?.artifacts ?? []);
    $: protectedWorkspace = workspacePath === DEFAULT_WORKSPACE_PATH || workspacePath === DEMO_WORKSPACE_PATH;

    onMount(async () => {
        const savedPath = localStorage.getItem("contextos_workspace_path");
        workspacePath = savedPath || getProject().workspacePath;
        openProject(workspacePath);
        await Promise.all([
            refreshSystemStatus(),
            hydrateWorkspaces(),
            refreshWorkspace(),
        ]);
    });

    async function refreshSystemStatus() {
        [apiStatus, workerStatus] = await Promise.all([
            probeService(API_URL),
            probeService("/worker"),
        ]);
        await checkCodexStatus();
    }

    async function checkCodexStatus() {
        try {
            const res = await fetch(`${API_URL}/codex/status`);
            if (!res.ok) {
                codexInstalled = false;
                codexLoggedIn = false;
                codexAccount = "";
                codexPlugins = [];
                return;
            }
            const body = await res.json();
            codexInstalled = body?.installed === true;
            codexLoggedIn = body?.logged_in === true;
            codexAccount = body?.account ?? "";
            codexPlugins = body?.plugins ?? [];
        } catch {
            codexInstalled = false;
            codexLoggedIn = false;
            codexAccount = "";
            codexPlugins = [];
        }
    }

    async function refreshWorkspace() {
        if (workspacePath === DEMO_WORKSPACE_PATH) {
            workspaceStatus = demoWorkspaceStatus();
            graphData = demoGraphData();
            activityArtifacts = demoArtifacts();
            lastFindings = demoFindings();
            lastAnalysisAt = "2026-01-01T09:30:00.000Z";
            return;
        }
        await loadWorkspaceStatus(workspacePath);
        workspaceStatus = await getWorkspaceStatus(workspacePath);
        graphData = await getGraphData(workspacePath);
        const artifacts = await getArtifacts({ workspace_id: workspacePath, limit: 12 });
        activityArtifacts = artifacts?.artifacts ?? [];
    }

    async function switchWorkspace(path: string) {
        if (!path) return;
        workspacePath = path;
        openProject(path);
        lastChatResult = null;
        lastFindings = null;
        await refreshWorkspace();
    }

    async function openDemoWorkspace() {
        walkthroughOpen = false;
        sourcePanelOpen = false;
        activeInsightTab = "findings";
        await switchWorkspace(DEMO_WORKSPACE_PATH);
    }

    async function createWorkspace() {
        const path = newWorkspacePath.trim();
        if (!path) return;
        addWorkspace(path);
        workspacePath = path;
        newWorkspacePath = "";
        sourcePanelOpen = true;
        lastChatResult = null;
        lastFindings = null;
        await refreshWorkspace();
    }

    function requestRemoveActiveWorkspace() {
        if (protectedWorkspace) return;
        workspacePendingRemoval = workspacePath;
        removeConfirmOpen = true;
    }

    function cancelWorkspaceRemoval() {
        if (removeInProgress) return;
        workspacePendingRemoval = "";
        removeConfirmOpen = false;
    }

    async function confirmWorkspaceRemoval() {
        if (removeInProgress) return;
        const path = workspacePendingRemoval || workspacePath;
        if (path === DEFAULT_WORKSPACE_PATH || path === DEMO_WORKSPACE_PATH) {
            cancelWorkspaceRemoval();
            return;
        }
        removeInProgress = true;
        try {
            const deleted = await deleteWorkspace(path);
            if (!deleted.ok) {
                addMessage(
                    assistantMsg(
                        `Workspace remove failed: backend delete did not complete${deleted.message ? `: ${deleted.message}` : "."}`,
                    ),
                );
                return;
            }
            removeWorkspace(path);
            workspacePath = getProject().workspacePath;
            newWorkspacePath = "";
            lastChatResult = null;
            lastFindings = null;
            graphData = null;
            selectedEntity = null;
            activityArtifacts = [];
            sourcePanelOpen = false;
            workspacePendingRemoval = "";
            removeConfirmOpen = false;
            await refreshWorkspace();
        } finally {
            removeInProgress = false;
        }
    }

    function makeId() {
        return Math.random().toString(36).slice(2) + Date.now().toString(36);
    }

    function now() {
        return new Date().toISOString();
    }

    function userMsg(text: string): ChatMessage {
        return { id: makeId(), role: "user", text, createdAt: now() };
    }

    function assistantMsg(
        text: string,
        card?: ChatMessage["card"],
    ): ChatMessage {
        return {
            id: makeId(),
            role: "assistant",
            text,
            createdAt: now(),
            card,
        };
    }

    function loadingMsg(text: string): ChatMessage {
        return {
            id: makeId(),
            role: "assistant",
            text,
            createdAt: now(),
            loading: true,
        };
    }

    function progressMsg(id: string, text: string): ChatMessage {
        return {
            id,
            role: "assistant",
            text,
            createdAt: now(),
            loading: true,
        };
    }

    async function submitCommand() {
        const text = command.trim();
        if (!text || busy) return;
        command = "";
        addMessage(userMsg(text));
        await routeCommand(text);
    }

    async function routeCommand(text: string) {
        const lower = text.toLowerCase();
        if (lower === "clear") {
            clearChat();
            lastChatResult = null;
            lastFindings = null;
            addMessage(assistantMsg("Chat history cleared for this workspace."));
            return;
        }
        if (
            lower.includes("install") ||
            lower.includes("setup") ||
            lower.includes("add source") ||
            lower.includes("connect source")
        ) {
            sourcePanelOpen = true;
            addMessage(assistantMsg("Source setup is open in the workspace panel."));
            return;
        }
        if (
            lower.includes("finding") ||
            lower.includes("mismatch") ||
            lower.startsWith("analyze") ||
            lower.startsWith("analyse")
        ) {
            await runFindings();
            return;
        }
        await runLocalQuery(text);
    }

    async function runLocalQuery(text: string) {
        const load = loadingMsg("Checking connected source context...");
        addMessage(load);
        busy = true;
        try {
            if (workspacePath === DEMO_WORKSPACE_PATH) {
                const result = demoChatQueryResult(text);
                lastChatResult = result;
                activityArtifacts = result.artifacts;
                replaceMessage(
                    load.id,
                    assistantMsg(result.answer, {
                        kind: "query",
                        chatResult: result,
                    }),
                );
                return;
            }

            const res = await postChatQuery({
                workspace_id: workspacePath,
                message: text,
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                local_date: localDateString(new Date()),
                limit: 20,
            });
            if (res.ok) {
                lastChatResult = res.body;
                replaceMessage(
                    load.id,
                    assistantMsg(res.body.answer, {
                        kind: "query",
                        chatResult: res.body,
                    }),
                );
                return;
            }
            replaceMessage(
                load.id,
                assistantMsg(
                    `Source query failed: ${res.body.message ?? res.body.error ?? "unknown error"}`,
                ),
            );
        } catch (error) {
            replaceMessage(
                load.id,
                assistantMsg(`Source query failed: ${String(error)}`),
            );
        } finally {
            busy = false;
            await refreshWorkspace();
        }
    }

    async function runFindings() {
        if (workspacePath === DEMO_WORKSPACE_PATH) {
            const findings = demoFindings();
            lastFindings = findings;
            lastAnalysisAt = new Date().toISOString();
            addMessage(
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

        const ready = getProject().connectors.filter(
            (source) => source.status === "ready",
        );
        if (ready.length === 0) {
            sourcePanelOpen = true;
            addMessage(
                assistantMsg(
                    "No ready sources in this workspace yet. Configure at least one source first.",
                ),
            );
            return;
        }

        const load = loadingMsg(`Running local analysis for ${ready.length} selected source${ready.length === 1 ? "" : "s"}...`);
        addMessage(load);
        busy = true;
        try {
            const codexConnectors = new Set<ConnectorKind>([
                "github",
                "jira",
                "slack",
                "notion",
                "sharepoint",
                "googledrive",
            ]);
            const completed: FindingsResult[] = [];
            const failures: FindingsFailure[] = [];
            const sourceStatuses: AnalysisSourceStatus[] = ready.map((source) => ({
                connector: source.connector,
                uri: source.uri,
                status: "queued",
            }));

            const updateProgress = () => {
                replaceMessage(load.id, progressMsg(load.id, buildAnalysisProgress(sourceStatuses)));
            };
            updateProgress();

            for (const [index, source] of ready.entries()) {
                const provider = codexConnectors.has(source.connector)
                    ? "codex"
                    : "token";
                sourceStatuses[index] = {
                    ...sourceStatuses[index],
                    status: "running",
                    detail: "request sent",
                };
                updateProgress();

                const controller = new AbortController();
                const timeout = window.setTimeout(() => controller.abort(), analysisSourceTimeoutMs);
                try {
                    const res = await postFindings({
                        workspace_id: workspacePath,
                        connector: source.connector,
                        uri: source.uri,
                        provider,
                        role: "pmo",
                        include_execution: false,
                    }, { signal: controller.signal });
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
                    const message = isAbortError(error)
                        ? `timed out after ${Math.round(analysisSourceTimeoutMs / 1000)}s`
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
            lastFindings = aggregated;
            lastAnalysisAt = new Date().toISOString();
            const summary = buildFindingsRunSummary({
                sourceCount: ready.length,
                completedCount: completed.length,
                result: aggregated,
                failures,
            });

            replaceMessage(
                load.id,
                assistantMsg(summary, {
                    kind: "findings",
                    findingsResult: aggregated ?? undefined,
                }),
            );
        } catch (error) {
            replaceMessage(
                load.id,
                assistantMsg(`Analysis failed: ${String(error)}`),
            );
        } finally {
            busy = false;
            await refreshWorkspace();
        }
    }

    function buildAnalysisProgress(statuses: AnalysisSourceStatus[]) {
        const done = statuses.filter((source) => source.status === "done").length;
        const failed = statuses.filter((source) => source.status === "failed").length;
        const lines = statuses.map((source, index) => {
            const label = `${index + 1}. ${source.connector}:${source.uri}`;
            if (source.status === "queued") return `${label} - queued`;
            if (source.status === "running") return `${label} - running`;
            if (source.status === "done") return `${label} - done${source.detail ? ` (${source.detail})` : ""}`;
            return `${label} - failed${source.detail ? `: ${source.detail}` : ""}`;
        });
        return `Running local analysis... ${done}/${statuses.length} complete, ${failed} failed.\n${lines.join("\n")}`;
    }

    function isAbortError(error: unknown) {
        return error instanceof DOMException && error.name === "AbortError";
    }

    async function handleKnowledgeDone() {
        sourcePanelOpen = false;
        await Promise.all([refreshWorkspace(), refreshSystemStatus()]);
        addMessage(assistantMsg("Source setup updated."));
    }

    function switchInsightTab(tab: "findings" | "graph" | "activity") {
        activeInsightTab = tab;
        sourcePanelOpen = false;
        if (tab === "graph" || tab === "activity") {
            void refreshWorkspace();
        }
    }

    function openGuideTarget(target: "sources" | "findings" | "graph" | "activity" | "agent") {
        if (target === "sources") {
            sourcePanelOpen = true;
            walkthroughOpen = false;
            return;
        }
        if (target === "agent") {
            sourcePanelOpen = false;
            walkthroughOpen = false;
            return;
        }
        switchInsightTab(target);
        walkthroughOpen = false;
    }

    function buildStatusLine(
        currentApiStatus: ServiceStatus,
        currentWorkerStatus: ServiceStatus,
        currentCodexInstalled: boolean,
        currentCodexLoggedIn: boolean,
        currentCodexAccount: string,
    ) {
        const api = serviceStatusText("API", currentApiStatus);
        const worker = serviceStatusText("Worker", currentWorkerStatus);
        const codex = currentApiStatus === "ok"
            ? !currentCodexInstalled
                ? "Codex CLI not installed"
                : !currentCodexLoggedIn
                    ? "Codex CLI not logged in"
                    : `Codex connected${currentCodexAccount ? ` as ${normalizeCodexAccount(currentCodexAccount)}` : ""}`
            : "Codex unavailable";
        return `${api} | ${worker} | ${codex}`;
    }

    function serviceStatusText(label: string, status: ServiceStatus) {
        if (status === "checking") return `${label}: checking`;
        if (status === "ok") return `${label}: online`;
        return `${label}: offline`;
    }

    function buildGraphLinks(entities: GraphEntity[], relationships: GraphRelationship[]) {
        if (relationships.length > 0) {
            return relationships
                .map((relationship) => ({
                    id: relationship.id,
                    source: relationship.from_id,
                    target: relationship.to_id,
                    label: relationship.kind,
                    strength: relationship.confidence ?? 0.5,
                    evidence: relationship.evidence,
                }))
                .sort((a, b) => b.strength - a.strength)
                .filter((link) => entities.some((entity) => entity.id === link.source) && entities.some((entity) => entity.id === link.target));
        }

        const links = new Map<string, GraphLink>();
        connectGroups(links, groupBy(entities, (entity) => entity.source || "unknown"), "source", 0.8, 4);
        connectGroups(links, groupBy(entities, (entity) => normalizedEvidence(entity)), "evidence", 0.55, 3);

        const aliasGroups = new Map<string, GraphEntity[]>();
        for (const entity of entities) {
            const aliases = [
                ...(entity.aliases ?? []),
                ...(entity.candidates ?? []).map((candidate) => candidate.alias),
            ];
            for (const alias of aliases) {
                const key = normalizeGraphKey(alias);
                if (!key) continue;
                aliasGroups.set(key, [...(aliasGroups.get(key) ?? []), entity]);
            }
        }
        connectGroups(links, aliasGroups, "alias", 0.95, 5);
        return [...links.values()]
            .sort((a, b) => b.strength - a.strength)
            .slice(0, 130);
    }

    function linkDegree(links: GraphLink[]) {
        const degree = new Map<string, number>();
        for (const link of links) {
            degree.set(link.source, (degree.get(link.source) ?? 0) + 1);
            degree.set(link.target, (degree.get(link.target) ?? 0) + 1);
        }
        return degree;
    }

    function connectGroups(
        links: Map<string, GraphLink>,
        groups: Map<string, GraphEntity[]>,
        label: string,
        strength: number,
        maxPerGroup: number,
    ) {
        for (const group of groups.values()) {
            const unique = dedupeEntities(group);
            if (unique.length < 2) continue;
            const sorted = unique
                .sort((a, b) => (b.confidence ?? 0) - (a.confidence ?? 0))
                .slice(0, maxPerGroup + 1);
            for (let index = 1; index < sorted.length; index += 1) {
                addGraphLink(links, sorted[0], sorted[index], label, strength);
            }
        }
    }

    function addGraphLink(
        links: Map<string, GraphLink>,
        source: GraphEntity,
        target: GraphEntity,
        label: string,
        strength: number,
    ) {
        if (source.id === target.id) return;
        const ids = [source.id, target.id].sort();
        const key = `${ids[0]}:${ids[1]}`;
        const existing = links.get(key);
        if (existing && existing.strength >= strength) return;
        links.set(key, { id: key, source: ids[0], target: ids[1], label, strength });
    }

    function groupBy(entities: GraphEntity[], keyFn: (entity: GraphEntity) => string) {
        const groups = new Map<string, GraphEntity[]>();
        for (const entity of entities) {
            const key = keyFn(entity);
            if (!key) continue;
            groups.set(key, [...(groups.get(key) ?? []), entity]);
        }
        return groups;
    }

    function dedupeEntities(entities: GraphEntity[]) {
        return [...new Map(entities.map((entity) => [entity.id, entity])).values()];
    }

    function normalizedEvidence(entity: GraphEntity) {
        return normalizeGraphKey(entity.evidence?.[0] ?? "");
    }

    function normalizeGraphKey(value: string) {
        return value.toLowerCase().replace(/[^a-z0-9]+/g, " ").trim().slice(0, 80);
    }

    function buildGraphLegendTypes(entities: GraphEntity[]): GraphTypeSummary[] {
        const counts = new Map<string, GraphTypeSummary>();
        for (const entity of entities) {
            const type = entity.type || "entity";
            const key = type.toLowerCase();
            const current = counts.get(key);
            if (current) {
                current.count += 1;
            } else {
                counts.set(key, { type, count: 1, color: graphTypeColor(type) });
            }
        }
        return [...counts.values()]
            .sort((a, b) => b.count - a.count || a.type.localeCompare(b.type));
    }

    function compareGraphEntities(a: GraphEntity & { degree: number }, b: GraphEntity & { degree: number }) {
        if (b.degree !== a.degree) return b.degree - a.degree;
        const confidenceDelta = (b.confidence ?? 0) - (a.confidence ?? 0);
        if (confidenceDelta !== 0) return confidenceDelta;
        return a.name.localeCompare(b.name);
    }

    function topGraphEntity(entities: GraphEntity[], degree: Map<string, number>) {
        return [...entities]
            .map((entity) => ({ ...entity, degree: degree.get(entity.id) ?? 0 }))
            .sort(compareGraphEntities)[0] ?? entities[0];
    }

    function linkedIdsForEntity(entityId: string, links: GraphLink[]) {
        const ids = new Set<string>();
        for (const link of links) {
            if (link.source === entityId) ids.add(link.target);
            if (link.target === entityId) ids.add(link.source);
        }
        return ids;
    }

    function buildEntityIndexSections(
        entities: GraphEntity[],
        degree: Map<string, number>,
        selected: GraphEntity | null,
        linkedIds: Set<string>,
        query: string,
    ): EntityIndexSection[] {
        const items = entities
            .map((entity) => ({ ...entity, degree: degree.get(entity.id) ?? 0 }))
            .sort(compareGraphEntities);
        const normalizedQuery = query.trim().toLowerCase();
        if (normalizedQuery) {
            return [
                {
                    label: "Matches",
                    entities: items
                        .filter((entity) =>
                            `${entity.name} ${entity.type}`.toLowerCase().includes(normalizedQuery),
                        )
                        .slice(0, 60),
                },
            ].filter((section) => section.entities.length > 0);
        }

        const used = new Set<string>();
        const sections: EntityIndexSection[] = [];
        if (selected) {
            const selectedItem = items.find((entity) => entity.id === selected.id);
            if (selectedItem) {
                sections.push({ label: "Selected", entities: [selectedItem] });
                used.add(selectedItem.id);
            }
        }

        const linked = items
            .filter((entity) => linkedIds.has(entity.id) && !used.has(entity.id))
            .slice(0, 14);
        if (linked.length) {
            sections.push({ label: "Linked", entities: linked });
            for (const entity of linked) used.add(entity.id);
        }

        const top = items
            .filter((entity) => !used.has(entity.id))
            .slice(0, Math.max(12, 36 - used.size));
        if (top.length) sections.push({ label: "Top entities", entities: top });
        return sections;
    }

    function buildFocusGraphRows(
        entity: GraphEntity,
        links: GraphLink[],
        entitiesById: Map<string, GraphEntity>,
    ): FocusGraphRow[] {
        const incoming = buildSideRows(entity, links, entitiesById, "incoming");
        const outgoing = buildSideRows(entity, links, entitiesById, "outgoing");
        return [...positionFocusRows(incoming), ...positionFocusRows(outgoing)];
    }

    function buildSideRows(
        entity: GraphEntity,
        links: GraphLink[],
        entitiesById: Map<string, GraphEntity>,
        side: "incoming" | "outgoing",
    ): FocusGraphRow[] {
        const rows: FocusGraphRow[] = [];
        for (const link of links) {
            if (side === "incoming" && link.target !== entity.id) continue;
            if (side === "outgoing" && link.source !== entity.id) continue;
            const otherId = side === "incoming" ? link.source : link.target;
            const other = entitiesById.get(otherId);
            if (!other) continue;
            rows.push({
                id: `${side}:${link.id}`,
                side,
                y: 50,
                entity: other,
                link,
                color: graphTypeColor(other.type || "entity"),
            });
        }
        return rows
            .sort((a, b) => b.link.strength - a.link.strength || a.entity.name.localeCompare(b.entity.name))
            .slice(0, 14);
    }

    function positionFocusRows(rows: FocusGraphRow[]) {
        if (rows.length === 0) return rows;
        const step = Math.min(11, 72 / Math.max(rows.length - 1, 1));
        const start = 50 - ((rows.length - 1) * step) / 2;
        return rows.map((row, index) => ({
            ...row,
            y: Math.max(12, Math.min(88, start + index * step)),
        }));
    }

    function graphTypeColor(type: string) {
        const palette = [
            "#1f5f8b",
            "#2d6a4f",
            "#b5523a",
            "#6f5aa8",
            "#8a6a20",
            "#2f7f7f",
            "#9b476e",
            "#59633a",
            "#7f4f2a",
            "#405f9a",
        ];
        let hash = 0;
        for (const char of type.toLowerCase()) {
            hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
        }
        return palette[hash % palette.length];
    }

    function typeAccentStyle(type: string) {
        return `--type-color:${graphTypeColor(type || "entity")};`;
    }

    function buildRelationshipGroups(
        entity: GraphEntity,
        links: GraphLink[],
        entitiesById: Map<string, GraphEntity>,
    ): RelationshipKindGroup[] {
        const groups = new Map<string, RelationshipKindGroup>();
        for (const link of links) {
            const kind = link.label || "related";
            const group = groups.get(kind) ?? { kind, incoming: [], outgoing: [] };
            const source = entitiesById.get(link.source);
            const target = entitiesById.get(link.target);
            if (link.source === entity.id && target) {
                group.outgoing.push({
                    entityName: target.name,
                    confidence: link.strength,
                    evidence: link.evidence,
                });
            } else if (link.target === entity.id && source) {
                group.incoming.push({
                    entityName: source.name,
                    confidence: link.strength,
                    evidence: link.evidence,
                });
            }
            groups.set(kind, group);
        }
        return [...groups.values()].sort((a, b) => a.kind.localeCompare(b.kind));
    }

    function formatTime(value?: string) {
        if (!value) return "never";
        return new Intl.DateTimeFormat(undefined, {
            month: "short",
            day: "2-digit",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit",
            timeZoneName: "short",
        }).format(new Date(value));
    }

    function findingDetectedTime() {
        return formatTime(lastAnalysisAt || new Date().toISOString());
    }

    function findingEvidenceTime() {
        const latest = recentArtifacts
            .map((artifact) => artifact.ingested_at)
            .filter(Boolean)
            .sort()
            .at(-1);
        return formatTime(latest || lastAnalysisAt || new Date().toISOString());
    }

    function severityLabel(value?: string) {
        const normalized = (value ?? "review").toLowerCase();
        if (normalized === "high") return "HIGH";
        if (normalized === "medium") return "MEDIUM";
        if (normalized === "low") return "LOW";
        return "REVIEW";
    }

    function relationshipLabel(value: string) {
        const normalized = value.replaceAll("_", " ");
        return normalized;
    }

    function findingSummary(mismatch: unknown) {
        const record = mismatch as Record<string, unknown>;
        return String(record.summary ?? record.mismatch_type ?? record.id ?? "Finding");
    }

    function findingDescription(mismatch: unknown) {
        const record = mismatch as Record<string, unknown>;
        return String(record.description ?? record.recommended_action ?? "Review this item against source evidence.");
    }

    function findingRecommendedAction(mismatch: unknown) {
        const record = mismatch as Record<string, unknown>;
        return String(record.recommended_action ?? "");
    }

    function findingImpact(mismatch: unknown) {
        const record = mismatch as Record<string, unknown>;
        return String(record.impact ?? "");
    }

    function localDateString(date: Date) {
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, "0");
        const day = String(date.getDate()).padStart(2, "0");
        return `${year}-${month}-${day}`;
    }

    function groupSources(sources: ConnectorKnowledge[]) {
        const groups = new Map<string, ConnectorKnowledge[]>();
        for (const source of sources) {
            const existing = groups.get(source.connector) ?? [];
            existing.push(source);
            groups.set(source.connector, existing);
        }
        return [...groups.entries()];
    }

    type MessageLine = {
        kind: "heading" | "number" | "bullet" | "body" | "blank";
        text: string;
    };

    function messageLines(text: string): MessageLine[] {
        return text.split("\n").map((line) => {
            const trimmed = line.trim();
            if (trimmed === "") return { kind: "blank", text: "" };
            if (/^\*\*[^*]+\*\*$/.test(trimmed)) {
                return { kind: "heading", text: cleanMarkdown(trimmed) };
            }
            if (/^\d+\.\s+/.test(trimmed)) {
                return { kind: "number", text: cleanMarkdown(trimmed) };
            }
            if (/^[-*]\s+/.test(trimmed)) {
                return { kind: "bullet", text: cleanMarkdown(trimmed.replace(/^[-*]\s+/, "")) };
            }
            return { kind: "body", text: cleanMarkdown(trimmed) };
        });
    }

    function cleanMarkdown(value: string) {
        return value.replace(/\*\*/g, "").replace(/`/g, "");
    }

    function previewText(value?: string, max = 360) {
        const text = cleanMarkdown((value ?? "").replace(/\s+/g, " ").trim());
        if (text.length <= max) return text;
        return `${text.slice(0, max).trim()}...`;
    }

    function normalizeCodexAccount(value: string) {
        const clean = value
            .split("\n")
            .map((line) => line.trim())
            .filter((line) => line && !line.toLowerCase().startsWith("warning:"))
            .slice(-1)[0];
        return clean || "Codex logged in";
    }

    function artifactOrigin(artifact: Artifact) {
        return artifact.connector === "filesystem" ? "LOCAL" : "SOURCE";
    }

    function artifactProvider(artifact: Artifact) {
        return artifact.connector === "filesystem" ? "Local file" : "Codex source";
    }

    function artifactSourceLabel(artifact: Artifact) {
        const metadata = artifact.metadata ?? {};
        if (artifact.connector === "slack") {
            return metadata.slack_channel_id || artifact.source_uri || artifact.title || "Slack";
        }
        if (artifact.connector === "github") {
            const owner = metadata.github_owner;
            const repo = metadata.github_repo;
            return owner && repo ? `${owner}/${repo}` : artifact.source_uri || artifact.title || "GitHub";
        }
        return artifact.source_uri || artifact.title || artifact.connector;
    }

    function artifactLink(artifact: Artifact) {
        const fields = [
            artifact.source_uri,
            artifact.metadata?.source_uri,
            artifact.metadata?.source_url,
            artifact.metadata?.url,
            artifact.body,
            artifact.preview,
        ];
        for (const field of fields) {
            const match = field?.match(/https?:\/\/[^\s)]+/);
            if (match) return match[0].replace(/[.,;]+$/, "");
        }
        return "";
    }

    function demoWorkspaceStatus(): WorkspaceStatus {
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
                { connector: "github", source_uri: "context-os/demo-api", event_count: 18, status: "ready" },
                { connector: "slack", source_uri: "#launch-review", event_count: 24, status: "ready" },
                { connector: "jira", source_uri: "DEMO", event_count: 12, status: "ready" },
            ],
        };
    }

    function demoFindings(): FindingsResult {
        return {
            connector: "multiple",
            uri: "3 demo sources",
            role: "pmo",
            trace_id: "demo-trace",
            summary: "Demo findings show how ContextOS connects source evidence into cross-layer delivery risks.",
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
                    description: "Jira says refund status must ship this sprint, but GitHub work only covers the UI state and Slack has an unresolved backend ownership question.",
                    evidence: ["jira:DEMO-42", "github:context-os/demo-api#18", "slack:#launch-review"],
                    confidence: 0.88,
                    impact: "PMO cannot confirm delivery readiness without backend ownership.",
                    recommended_action: "Assign a service owner and update the Jira acceptance criteria before release review.",
                },
                {
                    id: "demo-finding-2",
                    severity: "medium",
                    mismatch_type: "contract_drift",
                    summary: "API contract drift for refundStatus",
                    description: "Frontend discussion references refundStatus, while service notes still describe refund_state.",
                    evidence: ["github:context-os/demo-api#21", "slack:#launch-review"],
                    confidence: 0.76,
                    impact: "QA may validate the wrong response field.",
                    recommended_action: "Normalize the API contract name and add the field to the test plan.",
                },
            ],
        };
    }

    function demoGraphData(): GraphData {
        return {
            workspace_id: DEMO_WORKSPACE_PATH,
            count: 8,
            entity_count: 8,
            relationship_count: 7,
            entities: [
                { id: "entity-checkout", name: "Checkout", type: "feature", source: "jira", confidence: 0.94, evidence: ["DEMO-42 acceptance criteria"] },
                { id: "entity-refund", name: "Refund Status", type: "requirement", source: "jira", confidence: 0.9, evidence: ["DEMO-42: show refund status"] },
                { id: "entity-api", name: "Payments API", type: "service", source: "github", confidence: 0.86, evidence: ["context-os/demo-api#18"] },
                { id: "entity-ui", name: "Checkout UI", type: "presentation", source: "github", confidence: 0.82, evidence: ["context-os/demo-api#21"] },
                { id: "entity-qa", name: "QA Release Plan", type: "qa", source: "jira", confidence: 0.78, evidence: ["DEMO-51"] },
                { id: "entity-pmo", name: "Launch Review", type: "pmo", source: "slack", confidence: 0.84, evidence: ["#launch-review decision thread"] },
                { id: "entity-owner", name: "Service Owner", type: "person", source: "slack", confidence: 0.63, evidence: ["Ownership unresolved in Slack"] },
                { id: "entity-contract", name: "refundStatus contract", type: "contract", source: "github", confidence: 0.72, evidence: ["OpenAPI notes in PR #21"] },
            ],
            relationships: [
                { id: "rel-1", from_id: "entity-checkout", to_id: "entity-refund", kind: "requires", confidence: 0.94, evidence: ["DEMO-42"] },
                { id: "rel-2", from_id: "entity-refund", to_id: "entity-api", kind: "depends_on", confidence: 0.86, evidence: ["PR #18"] },
                { id: "rel-3", from_id: "entity-ui", to_id: "entity-contract", kind: "expects_contract", confidence: 0.82, evidence: ["PR #21"] },
                { id: "rel-4", from_id: "entity-contract", to_id: "entity-api", kind: "implemented_by", confidence: 0.72, evidence: ["API notes"] },
                { id: "rel-5", from_id: "entity-qa", to_id: "entity-contract", kind: "validates", confidence: 0.78, evidence: ["DEMO-51"] },
                { id: "rel-6", from_id: "entity-pmo", to_id: "entity-checkout", kind: "tracks", confidence: 0.84, evidence: ["Launch review"] },
                { id: "rel-7", from_id: "entity-owner", to_id: "entity-api", kind: "owns", confidence: 0.42, evidence: ["Unconfirmed Slack thread"] },
            ],
        };
    }

    function demoArtifacts(): Artifact[] {
        return [
            demoArtifact("demo-artifact-1", "jira", "DEMO-42", "Refund status acceptance criteria", "PMO asks for refund status in checkout before launch review.", "2026-01-01T09:25:00.000Z"),
            demoArtifact("demo-artifact-2", "github", "context-os/demo-api#18", "Payments API ownership question", "Backend PR covers API plumbing but does not assign a service owner.", "2026-01-01T09:15:00.000Z"),
            demoArtifact("demo-artifact-3", "slack", "#launch-review", "Launch review decision thread", "Team agrees the UI is ready but backend ownership is still unresolved.", "2026-01-01T09:20:00.000Z"),
            demoArtifact("demo-artifact-4", "github", "context-os/demo-api#21", "refundStatus naming drift", "Frontend uses refundStatus while service notes mention refund_state.", "2026-01-01T09:10:00.000Z"),
        ];
    }

    function demoChatQueryResult(text: string): ChatQueryResult {
        const lower = text.toLowerCase();
        const artifacts = demoArtifacts();
        let answer = "Demo workspace is working locally. It has Jira, GitHub, and Slack evidence saved for the same workspace, so you can inspect findings, graph, and recent activity without connecting real sources.";
        let summary = "Demo workspace status";
        let intent = "status";

        if (lower.includes("finding") || lower.includes("mismatch") || lower.includes("refund")) {
            intent = "findings";
            summary = "Demo refund status delivery risk";
            answer = "Jira says refund status must ship this sprint, GitHub currently covers the UI state, and Slack still has an unresolved backend ownership question. ContextOS flags that as a high-confidence requirement gap, with a second medium finding for refundStatus/refund_state contract drift.";
        } else if (lower.includes("graph") || lower.includes("entity") || lower.includes("relationship")) {
            intent = "artifacts";
            summary = "Demo graph evidence";
            answer = "The demo graph links Checkout, Refund Status, Payments API, Checkout UI, QA Release Plan, Launch Review, Service Owner, and the refundStatus contract. The weakest link is service ownership, which is why the finding appears.";
        } else if (lower.includes("source") || lower.includes("connected") || lower.includes("ingest")) {
            intent = "status";
            summary = "Demo source status";
            answer = "This demo workspace has 3 ready sources: Jira DEMO, GitHub context-os/demo-api, and Slack #launch-review. They are frontend demo records, so querying the demo does not call the backend workspace API.";
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

    function demoArtifact(id: string, connector: string, sourceURI: string, title: string, body: string, ingestedAt: string): Artifact {
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
</script>

<svelte:head>
    <title>ContextOS</title>
</svelte:head>

<main class="app-shell">
    <header class="topbar">
        <strong>CONTEXTOS</strong>
        <div class="workspace-control" title={workspacePath}>
            <select
                aria-label="Workspace"
                bind:value={workspacePath}
                on:change={(event) => switchWorkspace((event.currentTarget as HTMLSelectElement).value)}
            >
                {#each $workspaces as workspace (workspace.workspacePath)}
                    <option value={workspace.workspacePath}>{workspace.name}</option>
                {/each}
            </select>
            <form class="new-workspace" on:submit|preventDefault={createWorkspace}>
                <input bind:value={newWorkspacePath} placeholder="New workspace path" />
                <button type="submit" disabled={newWorkspacePath.trim() === ""}>New</button>
                <button
                    type="button"
                    on:click={requestRemoveActiveWorkspace}
                    disabled={busy || protectedWorkspace}
                    title={protectedWorkspace ? "Default and demo workspaces cannot be removed" : "Remove workspace"}
                >Remove</button>
            </form>
        </div>
        <div class="top-status">
            <button on:click={() => (sourcePanelOpen = true)}>Sources</button>
            <span
                class="status-chip"
                class:status-ok={apiStatus === "ok"}
                class:status-checking={apiStatus === "checking"}
                class:status-offline={apiStatus === "unreachable"}
                title="API status"
            >
                API {apiStatus === "ok" ? "Ready" : apiStatus === "checking" ? "Checking" : "Offline"}
            </span>
            <span
                class="status-chip"
                class:status-ok={workerStatus === "ok"}
                class:status-checking={workerStatus === "checking"}
                class:status-offline={workerStatus === "unreachable"}
                title="AI worker status"
            >
                Worker {workerStatus === "ok" ? "Ready" : workerStatus === "checking" ? "Checking" : "Offline"}
            </span>
            <span
                class="status-chip"
                class:status-ok={apiStatus === "ok" && codexInstalled && codexLoggedIn}
                class:status-checking={apiStatus === "checking"}
                class:status-offline={apiStatus !== "ok" || !codexInstalled || !codexLoggedIn}
                title="Codex status"
            >
                Codex {apiStatus !== "ok" ? "Unavailable" : !codexInstalled ? "Missing" : !codexLoggedIn ? "Login needed" : "Connected"}
            </span>
        </div>
    </header>

    <section class="main-grid">
        <section class="chat-pane" aria-label="Report agent chat">
            <section class="chat-card">
                <div class="chat-head">
                    <div>
                        <strong>Report Agent</strong>
                        <span>{hasSources ? "Ask against selected sources" : "Connect sources before asking"}</span>
                    </div>
                    <button on:click={() => clearChat()}>Clear</button>
                </div>

                <div class="messages" aria-live="polite">
                    {#if $chatMessages.length === 0}
                        <article class="message assistant">
                            <span>CONTEXT-OS</span>
                            <p>{hasSources ? "Ask about Slack messages, GitHub PRs, Jira tickets, docs, findings, or recent activity." : "Connect GitHub repos, Slack channels, or docs first."}</p>
                        </article>
                    {:else}
                        {#each $chatMessages as message (message.id)}
                            <article class="message" class:user={message.role === "user"}>
                                <span>{message.role === "user" ? "YOU" : "CONTEXT-OS"}</span>
                                {#if message.loading}
                                    <div class="message-body">
                                        {#each messageLines(message.text || "Working...") as line}
                                            {#if line.kind === "blank"}
                                                <div class="message-gap"></div>
                                            {:else if line.kind === "heading"}
                                                <h4>{line.text}</h4>
                                            {:else if line.kind === "number"}
                                                <p class="number-line">{line.text}</p>
                                            {:else if line.kind === "bullet"}
                                                <p class="bullet-line">{line.text}</p>
                                            {:else}
                                                <p>{line.text}</p>
                                            {/if}
                                        {/each}
                                    </div>
                                {:else}
                                    <div class="message-body">
                                        {#each messageLines(message.text) as line}
                                            {#if line.kind === "blank"}
                                                <div class="message-gap"></div>
                                            {:else if line.kind === "heading"}
                                                <h4>{line.text}</h4>
                                            {:else if line.kind === "number"}
                                                <p class="number-line">{line.text}</p>
                                            {:else if line.kind === "bullet"}
                                                <p class="bullet-line">{line.text}</p>
                                            {:else}
                                                <p>{line.text}</p>
                                            {/if}
                                        {/each}
                                    </div>
                                {/if}
                                {#if message.card?.chatResult?.artifacts?.length}
                                    <details>
                                        <summary>{message.card.chatResult.artifact_count} evidence items</summary>
                                        {#each message.card.chatResult.artifacts.slice(0, 5) as artifact (artifact.id)}
                                            {@const link = artifactLink(artifact)}
                                            <div class="evidence-item">
                                                <div class="evidence-meta">
                                                    <span>{artifact.connector}</span>
                                                    <small>{formatTime(artifact.ingested_at)}</small>
                                                </div>
                                                <strong>{artifactSourceLabel(artifact)}</strong>
                                                <div class="evidence-source-row">
                                                    {#if link}
                                                        <a href={link} target="_blank" rel="noreferrer">Open source</a>
                                                    {:else}
                                                        <span>{artifact.source_uri || "Stored local source"}</span>
                                                    {/if}
                                                </div>
                                            </div>
                                        {/each}
                                    </details>
                                {/if}
                                {#if message.card?.findingsResult?.mismatches?.length}
                                    <details>
                                        <summary>{message.card.findingsResult.mismatch_count ?? message.card.findingsResult.mismatches.length} findings</summary>
                                        {#each message.card.findingsResult.mismatches.slice(0, 5) as mismatch}
                                            <div class="evidence-item">
                                                <div class="finding-preview-head">
                                                    <span>{severityLabel(mismatch.severity)}</span>
                                                    <strong>{findingSummary(mismatch)}</strong>
                                                </div>
                                                <p>{findingDescription(mismatch)}</p>
                                                {#if findingImpact(mismatch)}
                                                    <p><b>Impact:</b> {findingImpact(mismatch)}</p>
                                                {/if}
                                                {#if findingRecommendedAction(mismatch)}
                                                    <p><b>Recommended:</b> {findingRecommendedAction(mismatch)}</p>
                                                {/if}
                                            </div>
                                        {/each}
                                    </details>
                                {/if}
                            </article>
                        {/each}
                    {/if}
                </div>

                <form class="composer" on:submit|preventDefault={submitCommand}>
                    <input
                        bind:value={command}
                        disabled={busy || !hasSources}
                        placeholder={hasSources ? "Ask about PRs, Slack threads, findings, or recent activity..." : "Connect sources first..."}
                    />
                    <button class="send-icon" aria-label="Send message" title="Send" disabled={busy || !hasSources || command.trim() === ""}>↑</button>
                </form>
            </section>
        </section>

        <section class="insight-pane" aria-label="Project insights">
            <section class="source-strip">
                <div>
                    <span>CODEX</span>
                    <strong>{codexLabel}</strong>
                </div>
                <div>
                    <span>WORKSPACE</span>
                    <strong>{$project.name}</strong>
                </div>
                <div>
                    <span>SOURCES</span>
                    <strong>{readySources.length}</strong>
                    <small>{codexLoggedIn ? "Codex connected" : "Codex login needed"}</small>
                </div>
            </section>

            {#if sourcePanelOpen}
                <section class="setup-panel">
                    <KnowledgeInstall
                        embedded
                        {codexLoggedIn}
                        {codexPlugins}
                        onClose={() => (sourcePanelOpen = false)}
                        on:done={handleKnowledgeDone}
                    />
                </section>
            {:else if hasSources}
                <section class="source-summary">
                    {#each sourceGroups as [connector, sources]}
                        <div>
                            <strong>{connector}</strong>
                            <span>{sources.length} selected</span>
                        </div>
                    {/each}
                </section>
            {:else}
                <section class="source-summary empty-source-summary">
                    <div>
                        <strong>No sources</strong>
                        <span>Use Sources to connect Codex plugins</span>
                    </div>
                </section>
            {/if}

            <section class="insight-card">
                <div class="insight-head">
                    <nav aria-label="Insight tabs">
                        <button type="button" class:active={activeInsightTab === "findings"} on:click={() => switchInsightTab("findings")}>Findings</button>
                        <button type="button" class:active={activeInsightTab === "graph"} on:click={() => switchInsightTab("graph")}>Graph</button>
                        <button type="button" class:active={activeInsightTab === "activity"} on:click={() => switchInsightTab("activity")}>Activity</button>
                    </nav>
                    <button type="button" on:click={runFindings} disabled={!hasSources || busy}>{busy ? "Running" : "Run Analysis"}</button>
                </div>

                {#if activeInsightTab === "findings"}
                    <div class="findings-view">
                        {#if lastFindings?.mismatches?.length}
                            {#each lastFindings.mismatches.slice(0, 6) as mismatch}
                                <article>
                                    <div class="finding-title-row">
                                        <span>{severityLabel(mismatch.severity)}</span>
                                        <strong>{findingSummary(mismatch)}</strong>
                                    </div>
                                    <div class="finding-time-row">
                                        <small>Detected: {findingDetectedTime()}</small>
                                        <small>Evidence: {findingEvidenceTime()}</small>
                                    </div>
                                    <div class="finding-copy">
                                        <p>{findingDescription(mismatch)}</p>
                                        {#if findingImpact(mismatch)}
                                            <p><b>Impact:</b> {findingImpact(mismatch)}</p>
                                        {/if}
                                    </div>
                                    {#if findingRecommendedAction(mismatch)}
                                        <div class="finding-action">
                                            <small>Recommended action</small>
                                            <p>{findingRecommendedAction(mismatch)}</p>
                                        </div>
                                    {/if}
                                </article>
                            {/each}
                        {:else if lastFindings}
                            <div class="empty-state">
                                <strong>Analysis ran, no mismatch signals detected</strong>
                                <p>Detected: {findingDetectedTime()}</p>
                                <p>Sources: {lastFindings.uri ?? readySources.length}. Events: {lastFindings.event_count ?? workspaceStatus?.event_count ?? 0}. Entities: {lastFindings.entity_count ?? workspaceStatus?.entity_count ?? 0}.</p>
                            </div>
                        {:else}
                            <div class="empty-state">
                                <strong>{hasSources ? "No findings yet" : "Connect sources to unlock findings"}</strong>
                                <p>{hasSources ? "Run analysis across selected sources to surface mismatches and delivery risks." : "Select GitHub repos, Slack channels, or docs first."}</p>
                            </div>
                        {/if}
                    </div>
                {:else if activeInsightTab === "graph"}
                    <div class="graph-workspace">
                        <div class="graph-canvas" aria-label="Typed entity map">
                            <div class="graph-count">
                                <strong>{graphEntities.length}</strong>
                                <span>entities | {graphLinks.length} links</span>
                            </div>

                            {#if graphEntities.length > 0}
                                <div class="graph-map-layout">
                                    <div class="entity-index" aria-label="Entity index grouped by type">
                                        <div class="entity-index-head">
                                            <strong>Entities</strong>
                                            <span>{entityIndexSections.reduce((sum, section) => sum + section.entities.length, 0)} shown</span>
                                        </div>
                                        <input
                                            class="entity-search"
                                            type="search"
                                            bind:value={entityQuery}
                                            placeholder="Filter entities"
                                            aria-label="Filter graph entities"
                                        />
                                        {#each entityIndexSections as section (section.label)}
                                            <section class="index-section">
                                                <h3>{section.label}</h3>
                                                <div class="entity-list">
                                                    {#each section.entities as entity (entity.id)}
                                                        <button
                                                            type="button"
                                                            class="entity-row"
                                                            class:selected={selectedEntity?.id === entity.id}
                                                            class:linked={selectedEntity !== null && selectedLinks.some((link) => link.source === entity.id || link.target === entity.id)}
                                                            style={typeAccentStyle(entity.type)}
                                                            on:click={() => (selectedEntity = entity)}
                                                        >
                                                            <span>{entity.name}</span>
                                                    <small>{entity.degree} link{entity.degree === 1 ? "" : "s"}</small>
                                                        </button>
                                                    {/each}
                                                </div>
                                            </section>
                                        {/each}
                                        {#if entityIndexSections.length === 0}
                                            <p class="entity-index-empty">No matching entities.</p>
                                        {/if}
                                    </div>

                                    <div class="focus-graph" aria-label="Selected entity relationship graph">
                                        {#if selectedEntity}
                                            <svg class="focus-lines" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                                                {#each focusRows as row (row.id)}
                                                    <path
                                                        d={row.side === "incoming"
                                                            ? `M 20 ${row.y} C 36 ${row.y}, 34 50, 48 50`
                                                            : `M 52 50 C 66 50, 64 ${row.y}, 80 ${row.y}`}
                                                        stroke={row.color}
                                                        class:strong={row.link.strength > 0.85}
                                                    />
                                                {/each}
                                            </svg>

                                            <div class="focus-column incoming">
                                                <strong>Incoming</strong>
                                                {#each incomingFocusRows as row (row.id)}
                                                    <button
                                                        type="button"
                                                        class="focus-node"
                                                        style={`top:${row.y}%;--type-color:${row.color};`}
                                                        on:click={() => (selectedEntity = row.entity)}
                                                    >
                                                        <span>{row.entity.name}</span>
                                                        <small>{relationshipLabel(row.link.label)}</small>
                                                    </button>
                                                {/each}
                                            </div>

                                            <button
                                                type="button"
                                                class="focus-center"
                                                style={typeAccentStyle(selectedEntity.type)}
                                                title={selectedEntity.name}
                                            >
                                                <span>{selectedEntity.type}</span>
                                                <strong>{selectedEntity.name}</strong>
                                                <small>{selectedLinks.length} link{selectedLinks.length === 1 ? "" : "s"}</small>
                                            </button>

                                            <div class="focus-column outgoing">
                                                <strong>Outgoing</strong>
                                                {#each outgoingFocusRows as row (row.id)}
                                                    <button
                                                        type="button"
                                                        class="focus-node"
                                                        style={`top:${row.y}%;--type-color:${row.color};`}
                                                        on:click={() => (selectedEntity = row.entity)}
                                                    >
                                                        <span>{row.entity.name}</span>
                                                        <small>{relationshipLabel(row.link.label)}</small>
                                                    </button>
                                                {/each}
                                            </div>

                                            {#if focusRows.length === 0}
                                                <div class="focus-empty">
                                                    <strong>No direct links</strong>
                                                    <p>Select another entity from the index to inspect relationships.</p>
                                                </div>
                                            {/if}
                                        {/if}
                                    </div>
                                </div>
                            {:else}
                                <div class="empty-graph">
                                    <strong>No graph data yet</strong>
                                    <p>{hasSources ? "Run analysis to populate local entities and relationships." : "Connect sources first, then run analysis to build the graph."}</p>
                                </div>
                            {/if}

                        </div>

                        <aside class="node-card">
                            {#if selectedEntity}
                                <div>
                                    <span>Node Details</span>
                                    <strong>{selectedEntity.type}</strong>
                                </div>
                                <p><b>Name:</b> {selectedEntity.name}</p>
                                <p><b>Links:</b> {graphDegree.get(selectedEntity.id) ?? 0}</p>
                                <p><b>Confidence:</b> {Math.round((selectedEntity.confidence ?? 0) * 100)}%</p>
                                <p><b>Source:</b> {selectedEntity.source || "unknown"}</p>
                                <hr />
                                {#if relationshipGroups.length}
                                    <div class="node-links">
                                        {#each relationshipGroups as group (group.kind)}
                                            <section>
                                                <h4>{relationshipLabel(group.kind)}</h4>
                                                {#if group.outgoing.length}
                                                    <small>Outgoing</small>
                                                    {#each group.outgoing as row}
                                                        <article>
                                                            <strong>{row.entityName}</strong>
                                                            <span>{Math.round(row.confidence * 100)}%</span>
                                                        </article>
                                                    {/each}
                                                {/if}
                                                {#if group.incoming.length}
                                                    <small>Incoming</small>
                                                    {#each group.incoming as row}
                                                        <article>
                                                            <strong>{row.entityName}</strong>
                                                            <span>{Math.round(row.confidence * 100)}%</span>
                                                        </article>
                                                    {/each}
                                                {/if}
                                            </section>
                                        {/each}
                                    </div>
                                    <hr />
                                {/if}
                                <p>{selectedEntity.evidence?.[0] ?? "Evidence appears after source ingestion and analysis."}</p>
                            {:else}
                                <div>
                                    <span>Node Details</span>
                                    <strong>none</strong>
                                </div>
                                <p>Select an entity row to inspect confidence, relationships, and source evidence.</p>
                            {/if}
                            {#if graphLegendTypes.length}
                                <section class="node-legend" aria-label="Entity types">
                                    <strong>Entity Types</strong>
                                    <div>
                                        {#each graphLegendTypes as item (item.type)}
                                            <span style={typeAccentStyle(item.type)}><i></i>{item.type} <b>{item.count}</b></span>
                                        {/each}
                                    </div>
                                </section>
                            {/if}
                        </aside>
                    </div>
                {:else}
                    <div class="activity-view">
                        {#if recentArtifacts.length}
                            {#each recentArtifacts.slice(0, 8) as artifact (artifact.id)}
                                <article>
                                    <div class="activity-meta">
                                        <span>{artifactOrigin(artifact)}</span>
                                        <small>{artifact.connector} | {artifactProvider(artifact)}</small>
                                    </div>
                                    <strong>{artifact.title || artifact.source_uri}</strong>
                                    <p>{previewText(artifact.preview)}</p>
                                    <small>{artifact.source_uri}</small>
                                    <small>{formatTime(artifact.ingested_at)}</small>
                                </article>
                            {/each}
                        {:else}
                            <div class="empty-state">
                                <strong>No activity loaded</strong>
                                <p>Ask chat about recent activity or run analysis after connecting sources.</p>
                            </div>
                        {/if}
                    </div>
                {/if}
            </section>

        </section>
    </section>

    <footer class="console-strip">
        <strong>{topContext}</strong>
        <span>{new Date().toLocaleTimeString()} | {statusLine}</span>
        <span>{graphData?.entity_count ?? graphData?.count ?? 0} graph nodes | {graphData?.relationship_count ?? graphLinks.length} links | {mismatchCount} findings</span>
    </footer>

    <button
        class="guide-fab"
        type="button"
        aria-expanded={walkthroughOpen}
        aria-label={walkthroughOpen ? "Close guide" : "Open guide"}
        title={walkthroughOpen ? "Close guide" : "Open guide"}
        on:click={() => (walkthroughOpen = !walkthroughOpen)}
    >
        <span aria-hidden="true">{walkthroughOpen ? "×" : "?"}</span>
    </button>

    {#if removeConfirmOpen}
        <div class="modal-backdrop">
            <div
                class="confirm-modal"
                role="dialog"
                aria-modal="true"
                aria-labelledby="remove-workspace-title"
                tabindex="-1"
            >
                <span>DELETE WORKSPACE</span>
                <h2 id="remove-workspace-title">Remove this workspace?</h2>
                <p>
                    This clears local chat, selected sources, source readiness, graph state, and project state for
                    <strong>{workspacePendingRemoval}</strong>. Backend workspace memory delete will be attempted too.
                </p>
                <div class="modal-actions">
                    <button type="button" on:click={cancelWorkspaceRemoval} disabled={removeInProgress}>Cancel</button>
                    <button class="danger" type="button" on:click={confirmWorkspaceRemoval} disabled={removeInProgress}>
                        {removeInProgress ? "Deleting" : "Delete"}
                    </button>
                </div>
            </div>
        </div>
    {/if}

    {#if walkthroughOpen}
        <div
            class="guide-popout"
            role="dialog"
            aria-modal="false"
            aria-labelledby="walkthrough-title"
            tabindex="-1"
        >
            <span>WALKTHROUGH</span>
            <h2 id="walkthrough-title">Open a section</h2>
            <div class="walkthrough-steps">
                <button type="button" on:click={openDemoWorkspace}>
                    <strong>Open Demo</strong>
                    <p>Switch to seeded demo data.</p>
                </button>
                <button type="button" on:click={() => openGuideTarget("sources")}>
                    <strong>Sources</strong>
                    <p>Connect or select source data.</p>
                </button>
                <button type="button" on:click={() => openGuideTarget("findings")}>
                    <strong>Findings</strong>
                    <p>Read issues, dates, evidence, and actions.</p>
                </button>
                <button type="button" on:click={() => openGuideTarget("graph")}>
                    <strong>Graph</strong>
                    <p>Inspect entities and relationships.</p>
                </button>
                <button type="button" on:click={() => openGuideTarget("activity")}>
                    <strong>Activity</strong>
                    <p>Review source artifacts behind the workspace.</p>
                </button>
                <button type="button" on:click={() => openGuideTarget("agent")}>
                    <strong>Report Agent</strong>
                    <p>Return to chat and ask against local data.</p>
                </button>
            </div>
        </div>
    {/if}
</main>

<style>
    :global(html),
    :global(body) {
        height: 100%;
        overflow: hidden;
    }

    :global(body) {
        margin: 0;
        background: #ebe8e0;
        color: #1c1b18;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        letter-spacing: 0;
    }

    :global(*) {
        box-sizing: border-box;
    }

    button,
    input,
    select {
        font: inherit;
    }

    button {
        cursor: pointer;
    }

    .app-shell {
        height: 100dvh;
        overflow: hidden;
        display: grid;
        grid-template-rows: 64px minmax(0, 1fr) 64px;
        background: #ebe8e0;
    }

    .topbar {
        display: grid;
        grid-template-columns: 128px minmax(0, 1fr) auto;
        align-items: center;
        border-bottom: 1px solid #d7d2c8;
        background: #ebe8e0;
        padding: 0 16px;
        gap: 14px;
    }

    .topbar strong,
    .topbar button,
    .top-status,
    .source-strip,
    .source-summary,
    .insight-head,
    .chat-head span,
    .message span,
    .console-strip {
        letter-spacing: 0.05em;
    }

    .topbar button,
    .new-workspace button,
    .insight-head button,
    .chat-head button,
    .composer button {
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background-color: transparent;
        background-image: linear-gradient(90deg, #1c1b18 0 50%, transparent 50% 100%);
        background-position: 100% 0;
        background-size: 200% 100%;
        color: #1c1b18;
        font-weight: 700;
        transition:
            background-position 0.18s ease,
            color 0.15s,
            border-color 0.15s,
            opacity 0.15s;
    }

    .topbar button:hover,
    .new-workspace button:hover:not(:disabled),
    .insight-head button:hover:not(:disabled),
    .chat-head button:hover,
    .composer button:hover:not(:disabled),
    .insight-head nav button.active {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .topbar button:disabled,
    .new-workspace button:disabled,
    .insight-head button:disabled,
    .chat-head button:disabled,
    .composer button:disabled {
        cursor: not-allowed;
        opacity: 0.42;
    }

    .workspace-control {
        min-width: 0;
        display: grid;
        grid-template-columns: minmax(220px, 340px) minmax(260px, 1fr);
        gap: 10px;
        align-items: center;
    }

    .workspace-control select,
    .new-workspace input {
        min-width: 0;
        height: 38px;
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background: transparent;
        color: #28261f;
        padding: 0 4px;
        font-size: 13px;
        font-weight: 700;
        outline: none;
    }

    .workspace-control select:focus,
    .new-workspace input:focus {
        border-bottom-color: #1c1b18;
        background: rgba(248, 246, 239, 0.7);
    }

    .new-workspace {
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto auto;
        gap: 6px;
        margin: 0;
    }

    .new-workspace button {
        height: 38px;
        min-width: 54px;
        padding: 0 10px;
        font-size: 11px;
    }

    .topbar button {
        height: 38px;
        min-width: 76px;
        padding: 0 12px;
        font-size: 12px;
    }

    .top-status {
        display: flex;
        align-items: center;
        justify-content: flex-end;
        flex-wrap: wrap;
        gap: 8px;
        color: #8a8678;
        font-size: 12px;
    }

    .status-chip {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        min-height: 38px;
        border-bottom: 1px solid #bdb7a8;
        padding: 0 12px 2px;
        white-space: nowrap;
    }

    .status-chip::before {
        content: "";
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: #8a8678;
        flex: 0 0 auto;
    }

    .status-ok {
        color: #2d6a4f;
    }

    .status-ok::before {
        background: #2d6a4f;
    }

    .status-checking {
        color: #8a6a20;
    }

    .status-checking::before {
        background: #8a6a20;
    }

    .status-offline {
        color: #b5523a;
    }

    .status-offline::before {
        background: #b5523a;
    }

    .main-grid {
        height: 100%;
        min-height: 0;
        display: grid;
        grid-template-columns: minmax(480px, 1fr) minmax(460px, 0.92fr);
        border-bottom: 1px solid #d7d2c8;
    }

    .chat-pane,
    .insight-pane {
        min-width: 0;
        min-height: 0;
        overflow: hidden;
    }

    .chat-pane {
        display: flex;
        min-height: 0;
        border-right: 1px solid #d7d2c8;
        padding: 16px;
    }

    .insight-pane {
        display: flex;
        flex-direction: column;
        gap: 10px;
        overflow: auto;
        padding: 14px 16px 16px;
        scrollbar-width: none;
    }

    .insight-pane::-webkit-scrollbar {
        display: none;
    }

    .graph-workspace {
        height: 100%;
        min-height: 0;
        display: grid;
        grid-template-columns: minmax(0, 1fr) 324px;
        gap: 0;
    }

    .graph-canvas {
        position: relative;
        min-height: 0;
        overflow: hidden;
        background: linear-gradient(180deg, #f1eee5, #ebe8e0);
        border-right: 1px solid #d7d2c8;
        padding: 16px;
    }

    .graph-map-layout {
        height: 100%;
        min-height: 520px;
        display: grid;
        grid-template-columns: 220px minmax(520px, 1fr);
        gap: 16px;
        padding-top: 0;
    }

    .entity-index {
        min-height: 0;
        max-height: calc(100vh - 230px);
        overflow: auto;
        scrollbar-width: none;
        display: flex;
        flex-direction: column;
        gap: 12px;
        padding-right: 2px;
        overscroll-behavior: contain;
    }

    .entity-index::-webkit-scrollbar,
    .messages::-webkit-scrollbar {
        display: none;
    }

    .entity-index-head {
        display: flex;
        align-items: baseline;
        justify-content: space-between;
        gap: 8px;
        border-bottom: 1px solid #d7d2c8;
        padding-bottom: 8px;
    }

    .entity-index-head strong,
    .index-section h3 {
        color: #1c1b18;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        font-size: 11px;
        text-transform: uppercase;
    }

    .entity-index-head span,
    .entity-index-empty {
        color: #8a8678;
        font-size: 11px;
    }

    .entity-search {
        width: 100%;
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background: transparent;
        color: #1c1b18;
        font: inherit;
        padding: 7px 0;
    }

    .entity-search:focus {
        border-bottom-color: #1c1b18;
        outline: none;
    }

    .index-section {
        min-width: 0;
    }

    .index-section h3 {
        margin: 0 0 5px;
        color: #d85d3f;
    }

    .graph-count {
        position: absolute;
        top: 14px;
        right: 14px;
        z-index: 5;
        display: flex;
        align-items: baseline;
        gap: 6px;
        border-bottom: 1px solid #bdb7a8;
        background: rgba(235, 232, 224, 0.82);
        padding: 6px 2px;
        color: #625f55;
        font-size: 11px;
        pointer-events: none;
    }

    .graph-count strong {
        color: #1c1b18;
    }

    .node-legend i {
        width: 9px;
        height: 9px;
        border-radius: 50%;
        background: var(--type-color);
        display: inline-block;
        flex: 0 0 auto;
    }

    .entity-list {
        display: grid;
        gap: 0;
    }

    .entity-row {
        min-width: 0;
        display: grid;
        grid-template-columns: minmax(0, 1fr);
        align-items: center;
        gap: 2px;
        border: 0;
        border-top: 1px solid rgba(215, 210, 200, 0.72);
        border-left: 3px solid transparent;
        background: transparent;
        color: #28261f;
        padding: 6px 0 6px 8px;
        text-align: left;
    }

    .entity-row:hover,
    .entity-row.selected {
        border-left-color: transparent;
        background: transparent;
    }

    .entity-row.linked:not(.selected) {
        border-left-color: transparent;
    }

    .entity-row span {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .entity-row small {
        color: #8a8678;
        font-size: 9px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .focus-graph {
        position: relative;
        min-width: 0;
        min-height: 520px;
        overflow: hidden;
        border: 1px solid rgba(215, 210, 200, 0.78);
        background:
            radial-gradient(circle, rgba(28, 27, 24, 0.09) 1px, transparent 1px) 0 0 / 22px 22px,
            rgba(248, 246, 239, 0.62);
    }

    .focus-lines {
        position: absolute;
        inset: 0;
        width: 100%;
        height: 100%;
        pointer-events: none;
    }

    .focus-lines path {
        fill: none;
        stroke-width: 1.4;
        stroke-opacity: 0.34;
        vector-effect: non-scaling-stroke;
    }

    .focus-lines path.strong {
        stroke-width: 2.2;
        stroke-opacity: 0.58;
    }

    .focus-center,
    .focus-node {
        position: absolute;
        z-index: 2;
        min-width: 0;
        border: 0;
        border-top: 1px solid rgba(215, 210, 200, 0.84);
        background: #f8f6ef;
        color: #1c1b18;
    }

    .focus-center {
        left: 50%;
        top: 50%;
        width: min(240px, 34%);
        min-height: 86px;
        display: grid;
        gap: 6px;
        transform: translate(-50%, -50%);
        border-top: 4px solid var(--type-color);
        border-bottom: 1px solid rgba(215, 210, 200, 0.84);
        padding: 14px;
        text-align: left;
    }

    .focus-center span,
    .focus-center small,
    .focus-node small,
    .focus-column > strong {
        color: #8a8678;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        font-size: 10px;
        text-transform: uppercase;
    }

    .focus-center strong {
        min-width: 0;
        overflow-wrap: anywhere;
        color: #1c1b18;
        font-size: 15px;
        line-height: 1.25;
    }

    .focus-column {
        position: absolute;
        top: 0;
        bottom: 0;
        width: 31%;
        pointer-events: none;
    }

    .focus-column.incoming {
        left: 14px;
    }

    .focus-column.outgoing {
        right: 14px;
    }

    .focus-column > strong {
        position: absolute;
        top: 12px;
        left: 0;
    }

    .focus-node {
        width: 100%;
        display: grid;
        gap: 4px;
        transform: translateY(-50%);
        border-left: 4px solid var(--type-color);
        padding: 8px 10px;
        text-align: left;
        pointer-events: auto;
    }

    .focus-node:hover {
        background: #fffdf7;
        border-left-color: var(--type-color);
    }

    .focus-node span {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .focus-empty {
        position: absolute;
        left: 50%;
        top: calc(50% + 92px);
        width: min(280px, 70%);
        transform: translateX(-50%);
        text-align: center;
        color: #625f55;
    }

    .empty-graph {
        position: absolute;
        border: 1px solid rgba(215, 210, 200, 0.65);
        border-radius: 8px;
        background: rgba(248, 246, 239, 0.92);
    }

    .node-card {
        min-height: 0;
        background: transparent;
        padding: 16px;
        font-size: 13px;
        overflow: auto;
    }

    .node-card div {
        display: flex;
        justify-content: space-between;
        margin-bottom: 12px;
    }

    .node-card span,
    .node-card strong {
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        font-size: 12px;
    }

    .node-card strong {
        color: #2d6a4f;
    }

    .node-card p {
        margin: 9px 0;
        line-height: 1.45;
        overflow-wrap: anywhere;
    }

    .node-card hr {
        border: 0;
        border-top: 1px solid #d7d2c8;
        margin: 14px 0;
    }

    .node-links {
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin: 0;
        padding: 0;
    }

    .node-links section {
        display: grid;
        gap: 6px;
        border-bottom: 1px solid #d7d2c8;
        background: transparent;
        padding: 0 0 10px;
    }

    .node-links h4 {
        margin: 0;
        color: #1c1b18;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        font-size: 11px;
        text-transform: uppercase;
    }

    .node-links small {
        color: #8a8678;
        font-size: 10px;
        text-transform: uppercase;
    }

    .node-links article {
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto;
        gap: 8px;
    }

    .node-links article strong {
        color: #1c1b18;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .node-links article span {
        color: #8a8678;
    }

    .node-legend {
        display: grid;
        gap: 8px;
        border-top: 1px solid #d7d2c8;
        margin-top: 14px;
        padding-top: 12px;
        color: #625f55;
        font-size: 12px;
    }

    .node-legend > strong {
        color: #d85d3f;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        font-size: 11px;
        text-transform: uppercase;
    }

    .node-legend div {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 7px 12px;
    }

    .node-legend span {
        display: inline-flex;
        align-items: center;
        min-width: 0;
        gap: 6px;
        text-transform: none;
        overflow: hidden;
        white-space: nowrap;
    }

    .node-legend b {
        color: #8a8678;
        font-weight: 400;
    }

    .empty-graph {
        left: 50%;
        top: 50%;
        transform: translate(-50%, -50%);
        padding: 18px;
        text-align: center;
    }

    .source-strip {
        display: grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: 10px;
        align-items: center;
        border-bottom: 1px solid #d7d2c8;
        padding: 4px 0 12px;
    }

    .source-strip span,
    .findings-view span,
    .activity-view span,
    .activity-view small {
        display: block;
        color: #8a8678;
        font-size: 11px;
        text-transform: uppercase;
    }

    .source-strip strong {
        display: block;
        margin-top: 4px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 13px;
    }

    .source-strip small {
        display: block;
        margin-top: 3px;
        color: #8a8678;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 11px;
    }

    .insight-head > button {
        padding: 8px 12px;
        white-space: nowrap;
    }

    .source-summary {
        display: flex;
        gap: 8px;
        overflow-x: auto;
        border-bottom: 1px solid #d7d2c8;
        padding: 0 0 10px;
    }

    .source-summary div {
        min-width: 130px;
        border-left: 2px solid #d7d2c8;
        padding: 2px 10px;
    }

    .source-summary strong,
    .source-summary span {
        display: block;
    }

    .source-summary strong {
        text-transform: uppercase;
        font-size: 12px;
    }

    .source-summary span {
        margin-top: 4px;
        color: #8a8678;
        font-size: 12px;
    }

    .empty-source-summary div {
        min-width: 100%;
    }

    .insight-card {
        min-height: 420px;
        flex: 1 0 420px;
        display: grid;
        grid-template-rows: auto minmax(0, 1fr);
        overflow: hidden;
        background: transparent;
    }

    .insight-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        border-bottom: 1px solid #d7d2c8;
        padding: 10px 0;
    }

    .insight-head nav {
        display: flex;
        gap: 4px;
        border-bottom: 1px solid #d7d2c8;
        border-radius: 0;
        background: transparent;
        padding: 4px;
    }

    .insight-head nav button {
        min-width: 86px;
        border: 0;
        border-bottom: 1px solid transparent;
        padding: 6px 10px;
        font-size: 12px;
    }

    .insight-head nav button.active {
        background-position: 0 0;
        color: #f8f6ef;
    }

    .findings-view,
    .activity-view {
        min-height: 0;
        display: flex;
        flex-direction: column;
        gap: 10px;
        overflow: auto;
        padding: 14px 0;
    }

    .findings-view article,
    .activity-view article,
    .empty-state {
        border-bottom: 1px solid #d7d2c8;
        padding: 14px 0;
    }

    .finding-title-row,
    .finding-preview-head {
        display: flex;
        align-items: baseline;
        gap: 10px;
        min-width: 0;
    }

    .finding-title-row span,
    .finding-preview-head span {
        flex: 0 0 auto;
        color: #d85d3f;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: 0.04em;
    }

    .finding-title-row strong,
    .finding-preview-head strong {
        min-width: 0;
        overflow-wrap: anywhere;
    }

    .finding-time-row {
        display: flex;
        flex-wrap: wrap;
        gap: 6px 14px;
        margin-top: 6px;
        padding-bottom: 8px;
        border-bottom: 1px solid rgba(215, 210, 200, 0.62);
    }

    .finding-copy,
    .finding-action {
        margin-top: 10px;
    }

    .finding-action {
        padding-left: 10px;
        border-left: 2px solid #d7d2c8;
    }

    .finding-action small {
        display: block;
        margin-bottom: 2px;
        font-weight: 700;
        letter-spacing: 0.03em;
        text-transform: uppercase;
    }

    .findings-view strong,
    .activity-view strong,
    .empty-state strong {
        display: block;
        margin-top: 0;
    }

    .findings-view p,
    .activity-view p,
    .empty-state p {
        margin: 6px 0 0;
        color: #5f5b50;
        line-height: 1.45;
    }

    .activity-meta {
        display: flex;
        align-items: baseline;
        justify-content: space-between;
        gap: 12px;
    }

    .activity-meta small {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    small {
        color: #8a8678;
    }

    .chat-card {
        flex: 1 1 auto;
        min-height: 280px;
        display: grid;
        grid-template-rows: auto minmax(0, 1fr) auto;
        overflow: hidden;
        background: transparent;
    }

    .chat-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
        border-bottom: 1px solid #d7d2c8;
        padding: 4px 0 12px;
    }

    .chat-head strong,
    .chat-head span {
        display: block;
    }

    .chat-head span {
        margin-top: 3px;
        color: #8a8678;
        font-size: 12px;
    }

    .chat-head button {
        padding: 7px 12px;
    }

    .messages {
        min-height: 0;
        display: flex;
        flex-direction: column;
        gap: 12px;
        overflow: auto;
        scrollbar-width: none;
        overscroll-behavior: contain;
        padding: 16px;
    }

    .message {
        width: min(680px, 90%);
        border-radius: 14px;
        background: transparent;
        padding: 4px 0;
        line-height: 1.5;
    }

    .message.user {
        align-self: flex-end;
        color: #1c1b18;
        padding: 4px 0;
        text-align: right;
    }

    .message span {
        display: block;
        margin-bottom: 6px;
        color: #8a8678;
        font-size: 12px;
    }

    .message p {
        margin: 0;
    }

    .message-body {
        display: grid;
        gap: 6px;
    }

    .message-body h4 {
        margin: 10px 0 2px;
        font-size: 13px;
        color: #28261f;
    }

    .message-body h4:first-child {
        margin-top: 0;
    }

    .message-body .number-line {
        margin-top: 8px;
        font-weight: 700;
    }

    .message-body .bullet-line {
        position: relative;
        padding-left: 14px;
    }

    .message-body .bullet-line::before {
        content: "";
        position: absolute;
        left: 0;
        top: 0.72em;
        width: 5px;
        height: 5px;
        border-radius: 50%;
        background: #8a8678;
    }

    .message-gap {
        height: 4px;
    }

    details {
        margin-top: 12px;
        border-top: 1px solid #d7d2c8;
        padding-top: 10px;
    }

    summary {
        cursor: pointer;
        font-weight: 700;
    }

    .evidence-item {
        margin-top: 10px;
        border: 0;
        border-top: 1px solid #d7d2c8;
        border-radius: 0;
        background: transparent;
        padding: 10px;
    }

    .evidence-meta,
    .evidence-source-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        min-width: 0;
    }

    .evidence-meta span {
        color: #d85d3f;
        font-size: 11px;
        font-weight: 700;
        text-transform: uppercase;
    }

    .evidence-item strong {
        display: block;
        margin: 7px 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .evidence-source-row a {
        color: #1f5f8b;
        font-weight: 700;
        text-decoration: none;
    }

    .evidence-source-row a:hover {
        color: #1c1b18;
        text-decoration: underline;
    }

    .evidence-source-row span {
        min-width: 0;
        overflow: hidden;
        color: #8a8678;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 12px;
    }

    .composer {
        display: grid;
        grid-template-columns: 1fr auto;
        gap: 10px;
        padding: 12px;
        border-top: 1px solid #d7d2c8;
    }

    .composer input {
        min-width: 0;
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background: transparent;
        padding: 11px 12px;
        outline: none;
    }

    .composer input:focus {
        border-bottom-color: #1c1b18;
    }

    .composer button {
        width: 44px;
        min-width: 44px;
        padding: 0;
        color: #1c1b18;
        font-size: 18px;
        font-weight: 700;
        line-height: 1;
    }

    .composer button:hover:not(:disabled) {
        color: #f8f6ef;
    }

    .setup-panel {
        max-height: none;
        overflow: visible;
        border-bottom: 1px solid #d7d2c8;
        padding: 0 12px 12px;
    }

    .console-strip {
        display: grid;
        align-content: center;
        gap: 6px;
        background: #070707;
        color: #d7d2c8;
        padding: 14px 12px;
        font-size: 11px;
    }

    .console-strip strong {
        color: #f8f6ef;
    }

    .guide-fab {
        position: fixed;
        right: 18px;
        bottom: 82px;
        z-index: 90;
        width: 44px;
        height: 44px;
        display: grid;
        place-items: center;
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 50%;
        background-color: #ebe8e0;
        background-image: linear-gradient(90deg, #1c1b18 0 50%, transparent 50% 100%);
        background-position: 100% 0;
        background-size: 200% 100%;
        color: #1c1b18;
        font-weight: 700;
        box-shadow: 0 10px 28px rgba(28, 27, 24, 0.14);
        padding: 0;
        transition:
            background-position 0.18s ease,
            color 0.15s,
            border-color 0.15s,
            opacity 0.15s;
    }

    .guide-fab span {
        display: block;
        margin-top: 0;
        font-size: 21px;
        line-height: 1;
    }

    .guide-fab:hover {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .modal-backdrop {
        position: fixed;
        inset: 0;
        z-index: 80;
        display: grid;
        place-items: center;
        background: rgba(28, 27, 24, 0.34);
        padding: 18px;
    }

    .confirm-modal {
        width: min(520px, 100%);
        border: 1px solid #1c1b18;
        background: #ebe8e0;
        box-shadow: 0 18px 48px rgba(28, 27, 24, 0.22);
        padding: 20px;
    }

    .confirm-modal > span {
        display: block;
        margin-bottom: 10px;
        color: #d85d3f;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: 0.05em;
    }

    .confirm-modal h2 {
        margin: 0 0 12px;
        color: #1c1b18;
        font-size: 18px;
        line-height: 1.25;
    }

    .confirm-modal p {
        margin: 0;
        color: #625f55;
        font-size: 13px;
        line-height: 1.6;
    }

    .confirm-modal p strong {
        color: #1c1b18;
        overflow-wrap: anywhere;
    }

    .guide-popout {
        position: fixed;
        right: 18px;
        bottom: 138px;
        z-index: 90;
        width: min(360px, calc(100vw - 36px));
        border: 1px solid #1c1b18;
        background: #ebe8e0;
        box-shadow: 0 18px 48px rgba(28, 27, 24, 0.22);
        padding: 16px;
    }

    .guide-popout > span {
        display: block;
        margin-bottom: 8px;
        color: #d85d3f;
        font-size: 10px;
        font-weight: 700;
        letter-spacing: 0.05em;
    }

    .guide-popout h2 {
        margin: 0 0 10px;
        color: #1c1b18;
        font-size: 16px;
        line-height: 1.25;
    }

    .walkthrough-steps {
        display: grid;
        gap: 0;
        margin-top: 8px;
        border-top: 1px solid #d7d2c8;
    }

    .walkthrough-steps button {
        width: 100%;
        border: 0;
        border-bottom: 1px solid #d7d2c8;
        border-radius: 0;
        background: transparent;
        padding: 10px 0;
        color: inherit;
        text-align: left;
        cursor: pointer;
    }

    .walkthrough-steps button:hover {
        background: transparent;
    }

    .walkthrough-steps strong {
        display: block;
        color: #1c1b18;
        font-size: 13px;
    }

    .walkthrough-steps p {
        margin-top: 5px;
        font-size: 12px;
        line-height: 1.55;
    }

    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 22px;
        border-top: 1px solid #d7d2c8;
        padding: 14px 0 4px;
    }

    .modal-actions button {
        min-width: 92px;
        height: 36px;
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background-color: transparent;
        background-image: linear-gradient(90deg, #1c1b18 0 50%, transparent 50% 100%);
        background-position: 100% 0;
        background-size: 200% 100%;
        color: #1c1b18;
        font-weight: 700;
        transition:
            background-position 0.18s ease,
            color 0.15s,
            border-color 0.15s,
            opacity 0.15s;
    }

    .modal-actions button:hover:not(:disabled) {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .modal-actions button.danger {
        background-image: linear-gradient(90deg, #d85d3f 0 50%, transparent 50% 100%);
        border-bottom-color: #d85d3f;
        color: #d85d3f;
    }

    .modal-actions button.danger:hover:not(:disabled) {
        background-position: 0 0;
        color: #fffdf7;
    }

    .modal-actions button:disabled {
        cursor: not-allowed;
        opacity: 0.45;
    }

    @media (max-width: 1100px) {
        .main-grid {
            grid-template-columns: 1fr;
        }

        .graph-workspace {
            grid-template-columns: 1fr;
            grid-template-rows: minmax(420px, 1fr) auto;
        }

        .graph-map-layout {
            grid-template-columns: 1fr;
            grid-template-rows: auto minmax(460px, 1fr);
        }

        .entity-index {
            max-height: 240px;
        }

        .node-card {
            max-height: 180px;
        }

        .chat-pane {
            border-right: none;
            border-bottom: 1px solid #d7d2c8;
        }
    }

    @media (max-width: 760px) {
        .topbar {
            grid-template-columns: 1fr;
            height: auto;
            gap: 8px;
            padding: 10px 16px;
        }

        .workspace-control {
            grid-template-columns: 1fr;
        }

        .top-status {
            justify-content: flex-start;
        }

        .graph-workspace {
            grid-template-rows: minmax(360px, 1fr) auto;
            padding: 8px;
        }

        .graph-canvas {
            padding: 10px;
        }

        .graph-map-layout {
            grid-template-rows: auto minmax(420px, 1fr);
            min-height: 660px;
            padding-top: 0;
        }

        .focus-graph {
            min-height: 420px;
        }

        .entity-index {
            max-height: 220px;
        }

        .focus-center {
            width: min(220px, 42%);
        }

        .focus-column {
            width: 34%;
        }

        .node-legend div {
            grid-template-columns: repeat(2, minmax(0, 1fr));
        }
    }
</style>
