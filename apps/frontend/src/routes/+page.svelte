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
    import KnowledgeInstall from "$lib/components/knowledge/KnowledgeInstall.svelte";

    type GraphNode = GraphEntity & {
        x: number;
        y: number;
        degree: number;
    };
    type GraphLink = {
        id: string;
        source: string;
        target: string;
        label: string;
        strength: number;
    };
    type GraphLegendType = {
        type: string;
        className: string;
        count: number;
    };

    let apiStatus: ServiceStatus = "checking";
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
    let graphNodes: GraphNode[] = [];
    let graphLinks: GraphLink[] = [];
    let graphLayoutKey = "";
    let graphCanvas: HTMLDivElement;
    let graphPan = { x: 0, y: 0 };
    let graphZoom = 1;
    let dragState:
        | { kind: "node"; id: string; lastX: number; lastY: number }
        | { kind: "pan"; startX: number; startY: number; originX: number; originY: number }
        | null = null;
    let lastChatResult: ChatQueryResult | null = null;
    let lastFindings: FindingsResult | null = null;
    let activityArtifacts: Artifact[] = [];

    $: readySources = $project.connectors.filter(
        (source) => source.status === "ready",
    );
    $: graphEntities = graphData?.entities ?? [];
    $: graphRelationships = graphData?.relationships ?? [];
    $: syncGraphLayout(graphEntities, graphRelationships);
    $: selectedLinks = selectedEntity
        ? graphLinks.filter((link) => link.source === selectedEntity?.id || link.target === selectedEntity?.id)
        : [];
    $: graphLegendTypes = buildGraphLegendTypes(graphNodes);
    $: if (
        graphEntities.length > 0 &&
        (!selectedEntity || !graphEntities.some((entity) => entity.id === selectedEntity?.id))
    ) {
        selectedEntity = graphEntities[0];
    }
    $: if (graphEntities.length === 0) {
        selectedEntity = null;
    }
    $: mismatchCount =
        workspaceStatus?.mismatch_count ?? lastFindings?.mismatch_count ?? 0;
    $: statusLine = buildStatusLine(
        apiStatus,
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
        apiStatus = await probeService(API_URL);
        await checkCodexStatus();
    }

    async function checkCodexStatus() {
        try {
            const res = await fetch(`${API_URL}/codex/status`);
            if (!res.ok) return;
            const body = await res.json();
            codexInstalled = body?.installed === true;
            codexLoggedIn = body?.logged_in === true;
            codexAccount = body?.account ?? "";
            codexPlugins = body?.plugins ?? [];
        } catch {
            codexInstalled = false;
            codexLoggedIn = false;
        }
    }

    async function refreshWorkspace() {
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

    async function removeActiveWorkspace() {
        if ($workspaces.length <= 1) return;
        removeWorkspace(workspacePath);
        workspacePath = getProject().workspacePath;
        newWorkspacePath = "";
        lastChatResult = null;
        lastFindings = null;
        await refreshWorkspace();
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
        const load = loadingMsg("Searching local source data...");
        addMessage(load);
        busy = true;
        try {
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
                    `Local query failed: ${res.body.message ?? res.body.error ?? "unknown error"}`,
                ),
            );
        } catch (error) {
            replaceMessage(
                load.id,
                assistantMsg(`Local query failed: ${String(error)}`),
            );
        } finally {
            busy = false;
            await refreshWorkspace();
        }
    }

    async function runFindings() {
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
            const failures: string[] = [];

            for (const source of ready) {
                const provider = codexConnectors.has(source.connector)
                    ? "codex"
                    : "token";
                const res = await postFindings({
                    workspace_id: workspacePath,
                    connector: source.connector,
                    uri: source.uri,
                    provider,
                    role: "pmo",
                    include_execution: false,
                });
                if (res.ok) {
                    completed.push(res.body);
                    lastFindings = res.body;
                } else {
                    failures.push(
                        `${source.connector}:${source.uri} - ${res.body.message ?? res.body.error ?? "unknown error"}`,
                    );
                }
            }

            const mismatchTotal = completed.reduce(
                (sum, result) => sum + (result.mismatch_count ?? result.mismatches?.length ?? 0),
                0,
            );
            const summary =
                `Analysis complete for ${completed.length}/${ready.length} selected source${ready.length === 1 ? "" : "s"}.` +
                ` Found ${mismatchTotal} finding${mismatchTotal === 1 ? "" : "s"}.` +
                (failures.length ? `\n\nFailed:\n- ${failures.join("\n- ")}` : "");

            replaceMessage(
                load.id,
                assistantMsg(summary, {
                    kind: "findings",
                    findingsResult: completed[completed.length - 1],
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

    function buildStatusLine(
        currentApiStatus: ServiceStatus,
        currentCodexInstalled: boolean,
        currentCodexLoggedIn: boolean,
        currentCodexAccount: string,
    ) {
        if (currentApiStatus === "checking") return "Checking API";
        if (currentApiStatus !== "ok") return "API offline";
        if (!currentCodexInstalled) return "Codex CLI not installed";
        if (!currentCodexLoggedIn) return "Codex CLI not logged in";
        return `Codex connected${currentCodexAccount ? ` as ${normalizeCodexAccount(currentCodexAccount)}` : ""}`;
    }

    function nodeStyle(node: GraphNode) {
        return `left:${node.x}%;top:${node.y}%;`;
    }

    function entityClass(entity: GraphEntity) {
        const value = entity.type.toLowerCase();
        if (value.includes("person")) return "person";
        if (value.includes("org") || value.includes("company")) return "org";
        if (value.includes("feature") || value.includes("project")) return "feature";
        return "entity";
    }

    function syncGraphLayout(entities: GraphEntity[], relationships: GraphRelationship[]) {
        const baseLinks = buildGraphLinks(entities, relationships);
        const degree = linkDegree(baseLinks);
        const visible = selectGraphEntities(entities, degree, 76);
        const visibleIds = new Set(visible.map((entity) => entity.id));
        const links = baseLinks
            .filter((link) => visibleIds.has(link.source) && visibleIds.has(link.target))
            .slice(0, 120);
        const key = `${visible.map((entity) => entity.id).join("|")}::${links.map((link) => link.id).join("|")}`;
        if (key === graphLayoutKey) return;

        const visibleDegree = linkDegree(links);
        const previous = new Map(graphNodes.map((node) => [node.id, node]));
        const nextNodes = layoutGraphNodes(visible, visibleDegree, previous);

        graphLayoutKey = key;
        graphLinks = links;
        graphNodes = nextNodes;
        graphPan = { x: 0, y: 0 };
        graphZoom = 1;
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
                }))
                .sort((a, b) => b.strength - a.strength)
                .slice(0, 180);
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

    function selectGraphEntities(entities: GraphEntity[], degree: Map<string, number>, limit: number) {
        return [...entities]
            .sort((a, b) => {
                const degreeDelta = (degree.get(b.id) ?? 0) - (degree.get(a.id) ?? 0);
                if (degreeDelta !== 0) return degreeDelta;
                return (b.confidence ?? 0) - (a.confidence ?? 0);
            })
            .slice(0, limit);
    }

    function layoutGraphNodes(
        entities: GraphEntity[],
        degree: Map<string, number>,
        previous: Map<string, GraphNode>,
    ) {
        const groups = groupGraphEntities(entities);
        const centers = clusterCenters(groups.length);
        const nodes: GraphNode[] = [];

        groups.forEach((group, groupIndex) => {
            const center = centers[groupIndex] ?? { x: 50, y: 48 };
            const sorted = [...group.entities].sort(
                (a, b) => (degree.get(b.id) ?? 0) - (degree.get(a.id) ?? 0),
            );
            sorted.forEach((entity, index) => {
                const existing = previous.get(entity.id);
                if (existing) {
                    nodes.push({ ...entity, x: existing.x, y: existing.y, degree: degree.get(entity.id) ?? 0 });
                    return;
                }
                const ring = Math.floor(index / 10);
                const position = index % 10;
                const radius = index === 0 ? 0 : 8 + ring * 7;
                const angle = position * 2.399963 + groupIndex * 0.65;
                nodes.push({
                    ...entity,
                    x: clamp(center.x + Math.cos(angle) * radius, 7, 93),
                    y: clamp(center.y + Math.sin(angle) * radius, 10, 88),
                    degree: degree.get(entity.id) ?? 0,
                });
            });
        });

        return nodes;
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

    function groupGraphEntities(entities: GraphEntity[]) {
        const groups = [...groupBy(entities, (entity) => entity.type || "unknown").entries()]
            .map(([key, groupEntities]) => ({ key, entities: groupEntities }))
            .sort((a, b) => b.entities.length - a.entities.length);
        return groups.length ? groups : [{ key: "graph", entities }];
    }

    function clusterCenters(count: number) {
        if (count <= 1) return [{ x: 50, y: 45 }];
        const centers: { x: number; y: number }[] = [];
        for (let index = 0; index < count; index += 1) {
            const angle = (index / Math.max(count, 1)) * Math.PI * 2 - Math.PI / 2;
            centers.push({
                x: 50 + Math.cos(angle) * 26,
                y: 46 + Math.sin(angle) * 22,
            });
        }
        return centers;
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

    function graphNodeById(id: string) {
        return graphNodes.find((node) => node.id === id);
    }

    function shouldShowNodeLabel(node: GraphNode) {
        return node.degree >= 3 || selectedEntity?.id === node.id;
    }

    function buildGraphLegendTypes(nodes: GraphNode[]): GraphLegendType[] {
        const counts = new Map<string, { type: string; count: number; className: string }>();
        for (const node of nodes) {
            const type = node.type || "entity";
            const key = type.toLowerCase();
            const current = counts.get(key);
            if (current) {
                current.count += 1;
            } else {
                counts.set(key, { type, count: 1, className: entityClass(node) });
            }
        }
        return [...counts.values()]
            .sort((a, b) => b.count - a.count)
            .slice(0, 6);
    }

    function clamp(value: number, min: number, max: number) {
        return Math.min(max, Math.max(min, value));
    }

    function startNodeDrag(event: PointerEvent, node: GraphNode) {
        event.preventDefault();
        event.stopPropagation();
        selectedEntity = node;
        dragState = { kind: "node", id: node.id, lastX: node.x, lastY: node.y };
    }

    function startGraphPan(event: PointerEvent) {
        if (event.button !== 0 || (event.target as HTMLElement).closest(".node-card, .legend")) return;
        dragState = {
            kind: "pan",
            startX: event.clientX,
            startY: event.clientY,
            originX: graphPan.x,
            originY: graphPan.y,
        };
    }

    function handleGraphPointerMove(event: PointerEvent) {
        if (!dragState || !graphCanvas) return;
        if (dragState.kind === "pan") {
            graphPan = {
                x: clamp(dragState.originX + event.clientX - dragState.startX, -260, 260),
                y: clamp(dragState.originY + event.clientY - dragState.startY, -180, 180),
            };
            return;
        }

        const nodeId = dragState.id;
        const rect = graphCanvas.getBoundingClientRect();
        const x = ((event.clientX - rect.left - graphPan.x) / (rect.width * graphZoom)) * 100;
        const y = ((event.clientY - rect.top - graphPan.y) / (rect.height * graphZoom)) * 100;
        const nextX = clamp(x, 4, 96);
        const nextY = clamp(y, 6, 94);
        const dx = nextX - dragState.lastX;
        const dy = nextY - dragState.lastY;
        const linkedIds = linkedNodeIds(nodeId);
        graphNodes = graphNodes.map((node) =>
            node.id === nodeId
                ? { ...node, x: nextX, y: nextY }
                : linkedIds.has(node.id)
                    ? {
                            ...node,
                            x: clamp(node.x + dx * 0.42, 4, 96),
                            y: clamp(node.y + dy * 0.42, 6, 94),
                        }
                : node,
        );
        dragState = { ...dragState, lastX: nextX, lastY: nextY };
    }

    function stopGraphDrag() {
        dragState = null;
    }

    function linkedNodeIds(nodeId: string) {
        const ids = new Set<string>();
        for (const link of graphLinks) {
            if (link.source === nodeId) ids.add(link.target);
            if (link.target === nodeId) ids.add(link.source);
        }
        return ids;
    }

    function zoomGraph(delta: number) {
        graphZoom = clamp(Number((graphZoom + delta).toFixed(2)), 0.55, 2.4);
    }

    function resetGraphView() {
        graphPan = { x: 0, y: 0 };
        graphZoom = 1;
    }

    function handleGraphWheel(event: WheelEvent) {
        event.preventDefault();
        const delta = event.deltaY > 0 ? -0.12 : 0.12;
        zoomGraph(delta);
    }

    function formatTime(value?: string) {
        if (!value) return "never";
        return new Intl.DateTimeFormat("en", {
            month: "short",
            day: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
        }).format(new Date(value));
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
</script>

<svelte:head>
    <title>ContextOS</title>
</svelte:head>

<svelte:window on:pointermove={handleGraphPointerMove} on:pointerup={stopGraphDrag} />

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
                <button type="button" on:click={removeActiveWorkspace} disabled={$workspaces.length <= 1}>Remove</button>
            </form>
        </div>
        <div class="top-status">
            <button on:click={() => (sourcePanelOpen = true)}>Sources</button>
            <strong>{apiStatus === "ok" ? "Ready" : "Offline"}</strong>
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
                            <p>{hasSources ? "Ask about Slack messages, GitHub PRs, Jira tickets, docs, findings, or recent activity." : "Connect GitHub repos, Slack channels, or docs first. After setup, chat will answer from those selected sources."}</p>
                        </article>
                    {:else}
                        {#each $chatMessages as message (message.id)}
                            <article class="message" class:user={message.role === "user"}>
                                <span>{message.role === "user" ? "YOU" : "CONTEXT-OS"}</span>
                                {#if message.loading}
                                    <p>Working...</p>
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
                                                <strong>{mismatch.severity ?? "review"}</strong>
                                                <p>{mismatch.description ?? mismatch.mismatch_type ?? mismatch.id}</p>
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
                    {#if !hasSources}
                        <div class="sample-sources">
                            <div class="sample-head">
                                <strong>Checklist preview</strong>
                                <span>Real repos and channels will load from connected accounts.</span>
                            </div>
                            <label><input type="checkbox" disabled /> GitHub / context-os</label>
                            <label><input type="checkbox" disabled /> GitHub / docs-site</label>
                            <label><input type="checkbox" disabled /> Slack / #engineering</label>
                            <label><input type="checkbox" disabled /> Slack / #product</label>
                        </div>
                    {/if}
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
                                    <span>{mismatch.severity ?? "review"}</span>
                                    <strong>{mismatch.summary ?? mismatch.mismatch_type ?? mismatch.id ?? "Finding"}</strong>
                                    <p>{mismatch.description ?? mismatch.recommended_action ?? "Review this item against source evidence."}</p>
                                </article>
                            {/each}
                        {:else}
                            <div class="empty-state">
                                <strong>{hasSources ? "No findings yet" : "Connect sources to unlock findings"}</strong>
                                <p>{hasSources ? "Run analysis across selected sources to surface mismatches and delivery risks." : "Select GitHub repos, Slack channels, or docs first."}</p>
                            </div>
                        {/if}
                    </div>
                {:else if activeInsightTab === "graph"}
                    <div class="graph-workspace">
                        <div
                            class="graph-canvas"
                            class:dragging={dragState !== null}
                            bind:this={graphCanvas}
                            role="application"
                            aria-label="Draggable entity graph"
                            on:pointerdown={startGraphPan}
                            on:wheel={handleGraphWheel}
                        >
                            <div class="graph-tools" aria-label="Graph zoom controls">
                                <button type="button" on:click={() => zoomGraph(0.18)}>+</button>
                                <button type="button" on:click={() => zoomGraph(-0.18)}>-</button>
                                <button type="button" on:click={resetGraphView}>{Math.round(graphZoom * 100)}%</button>
                            </div>
                            <div class="graph-count">
                                <strong>{graphNodes.length}</strong>
                                <span>shown of {graphEntities.length}</span>
                            </div>

                            <div
                                class="graph-plane"
                                style={`transform: translate(${graphPan.x}px, ${graphPan.y}px) scale(${graphZoom});`}
                            >
                                <svg class="graph-links" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                                    {#each graphLinks as link (link.id)}
                                        {@const source = graphNodeById(link.source)}
                                        {@const target = graphNodeById(link.target)}
                                        {#if source && target}
                                            <line
                                                x1={source.x}
                                                y1={source.y}
                                                x2={target.x}
                                                y2={target.y}
                                                class:strong={link.strength > 0.85}
                                            />
                                        {/if}
                                    {/each}
                                </svg>

                                {#each graphNodes as entity (entity.id)}
                                    <button
                                        type="button"
                                        class={`node ${entityClass(entity)}`}
                                        class:selected={selectedEntity?.id === entity.id}
                                        class:connected={entity.degree > 0}
                                        class:labeled={shouldShowNodeLabel(entity)}
                                        style={nodeStyle(entity)}
                                        title={`${entity.name}${entity.degree ? ` · ${entity.degree} links` : ""}`}
                                        on:pointerdown={(event) => startNodeDrag(event, entity)}
                                        on:click={() => (selectedEntity = entity)}
                                    >
                                        <span></span>
                                        <em>{entity.name}</em>
                                    </button>
                                {/each}
                            </div>

                            {#if graphEntities.length === 0}
                                <div class="empty-graph">
                                    <strong>No graph data yet</strong>
                                    <p>{hasSources ? "Run analysis to populate local entities and relationships." : "Connect sources first, then run analysis to build the graph."}</p>
                                </div>
                            {/if}

                            <div class="legend">
                                <strong>ENTITY TYPES</strong>
                                {#each graphLegendTypes as item (item.type)}
                                    <span><i class={item.className}></i>{item.type} <b>{item.count}</b></span>
                                {/each}
                            </div>
                        </div>

                        <aside class="node-card">
                            {#if selectedEntity}
                                <div>
                                    <span>Node Details</span>
                                    <strong>{selectedEntity.type}</strong>
                                </div>
                                <p><b>Name:</b> {selectedEntity.name}</p>
                                <p><b>Links:</b> {graphNodeById(selectedEntity.id)?.degree ?? 0}</p>
                                <p><b>Confidence:</b> {Math.round((selectedEntity.confidence ?? 0) * 100)}%</p>
                                <hr />
                                {#if selectedLinks.length}
                                    <ul class="node-links">
                                        {#each selectedLinks.slice(0, 6) as link (link.id)}
                                            {@const other = graphNodeById(link.source === selectedEntity.id ? link.target : link.source)}
                                            <li>
                                                <span class="relationship-kind">{link.label.replaceAll("_", " ")}</span>
                                                <strong>{other?.name ?? "Unknown"}</strong>
                                            </li>
                                        {/each}
                                    </ul>
                                    <hr />
                                {/if}
                                <p>{selectedEntity.evidence?.[0] ?? "Evidence appears after source ingestion and analysis."}</p>
                            {:else}
                                <div>
                                    <span>Node Details</span>
                                    <strong>none</strong>
                                </div>
                                <p>Select or drag a node to inspect its confidence, links, and source evidence.</p>
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
        grid-template-rows: 64px minmax(0, 1fr) 50px;
        background: #ebe8e0;
    }

    .topbar {
        display: grid;
        grid-template-columns: 128px minmax(0, 1fr) auto;
        align-items: center;
        border-bottom: 1px solid #d7d2c8;
        background: rgba(248, 246, 239, 0.9);
        padding: 0 12px;
        gap: 14px;
    }

    .topbar strong,
    .topbar button,
    .context-switcher,
    .top-status,
    .source-strip,
    .source-summary,
    .insight-head,
    .sample-sources,
    .chat-head span,
    .message span,
    .console-strip {
        letter-spacing: 0.05em;
    }

    .topbar button,
    .new-workspace button,
    .insight-head button,
    .chat-head button,
    .composer button,
    .graph-tools button {
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background: #f8f6ef;
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
    .graph-tools button:hover:not(:disabled),
    .insight-head nav button.active {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .topbar button:disabled,
    .new-workspace button:disabled,
    .insight-head button:disabled,
    .chat-head button:disabled,
    .composer button:disabled,
    .graph-tools button:disabled {
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
        min-width: 76px;
        padding: 5px 12px;
        font-size: 12px;
    }

    .top-status {
        display: flex;
        align-items: center;
        justify-content: flex-end;
        gap: 8px;
        color: #8a8678;
        font-size: 12px;
    }

    .top-status strong {
        color: #2d6a4f;
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
    }

    .graph-workspace {
        height: 100%;
        min-height: 0;
        display: grid;
        grid-template-columns: minmax(0, 1fr) 304px;
        gap: 0;
    }

    .graph-canvas {
        position: relative;
        min-height: 0;
        overflow: hidden;
        cursor: grab;
        touch-action: none;
        background:
            radial-gradient(circle, rgba(28, 27, 24, 0.13) 1px, transparent 1px) 0 0 / 22px 22px,
            linear-gradient(180deg, #f1eee5, #ebe8e0);
        border-right: 1px solid #d7d2c8;
    }

    .graph-canvas.dragging {
        cursor: grabbing;
    }

    .graph-plane {
        position: absolute;
        inset: 0;
        transform-origin: 0 0;
        transition: transform 120ms ease;
    }

    .graph-canvas.dragging .graph-plane {
        transition: none;
    }

    .graph-links {
        position: absolute;
        inset: 0;
        width: 100%;
        height: 100%;
        overflow: visible;
        pointer-events: none;
    }

    .graph-links line {
        stroke: rgba(31, 95, 139, 0.34);
        stroke-width: 1.45;
        vector-effect: non-scaling-stroke;
    }

    .graph-links line.strong {
        stroke: rgba(216, 93, 63, 0.54);
        stroke-width: 2.1;
    }

    .graph-tools {
        position: absolute;
        top: 12px;
        left: 12px;
        z-index: 5;
        display: flex;
        gap: 4px;
        background: rgba(248, 246, 239, 0.88);
        padding: 2px 0;
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

    .graph-tools button {
        min-width: 34px;
        height: 30px;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        font-size: 12px;
    }

    .node {
        position: absolute;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        border: 0;
        border-radius: 999px;
        background: transparent;
        color: #535047;
        cursor: grab;
        padding: 4px;
        transform: translate(-50%, -50%);
        user-select: none;
    }

    .node:hover,
    .node.selected {
        color: #1c1b18;
        z-index: 3;
    }

    .node:hover {
        background: rgba(248, 246, 239, 0.94);
        box-shadow: 0 8px 22px rgba(28, 27, 24, 0.1);
    }

    .node:hover em,
    .node.labeled em,
    .node.selected em {
        display: inline-block;
    }

    .node.selected em {
        font-weight: 700;
    }

    .node span,
    .legend i {
        width: 9px;
        height: 9px;
        border-radius: 50%;
        background: #1f5f8b;
        display: inline-block;
        flex: 0 0 auto;
    }

    .node.connected span {
        box-shadow: 0 0 0 3px rgba(31, 95, 139, 0.14);
    }

    .node.selected span {
        box-shadow: 0 0 0 4px rgba(216, 93, 63, 0.2);
    }

    .node em {
        display: none;
        max-width: 132px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        border: 1px solid rgba(215, 210, 200, 0.8);
        border-radius: 999px;
        background: rgba(248, 246, 239, 0.96);
        padding: 3px 7px;
        color: #28261f;
        font-size: 10px;
        font-style: normal;
    }

    .node.org span,
    .legend .org {
        background: #2d6a4f;
    }

    .node.person span,
    .legend .person {
        background: #d85d3f;
    }

    .node.feature span,
    .legend .feature {
        background: #8a8678;
    }

    .legend,
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
    .node-card strong,
    .legend strong {
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
        gap: 8px;
        margin: 0;
        padding: 0;
        list-style: none;
    }

    .node-links li {
        display: grid;
        gap: 2px;
        border-bottom: 1px solid #d7d2c8;
        background: transparent;
        padding: 8px 0;
    }

    .node-links li span {
        color: #8a8678;
        text-transform: uppercase;
    }

    .node-links li strong {
        color: #1c1b18;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .legend {
        left: 18px;
        bottom: 18px;
        display: grid;
        grid-template-columns: repeat(2, minmax(0, auto));
        gap: 8px 14px;
        padding: 14px;
        color: #625f55;
        font-size: 12px;
    }

    .legend strong {
        grid-column: 1 / -1;
        color: #d85d3f;
    }

    .legend span {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        text-transform: none;
        white-space: nowrap;
    }

    .legend b {
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
    .sample-head span,
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

    .sample-sources {
        display: grid;
        gap: 8px;
        margin-top: 10px;
        border-top: 1px solid #d7d2c8;
        padding: 12px 0 0;
    }

    .sample-head {
        display: flex;
        justify-content: space-between;
        gap: 12px;
    }

    .sample-sources label {
        display: flex;
        gap: 8px;
        align-items: center;
        color: #535047;
        font-size: 13px;
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
        padding: 12px 0;
    }

    .findings-view strong,
    .activity-view strong,
    .empty-state strong {
        display: block;
        margin-top: 4px;
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

    .muted,
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
        border: 1px solid #ece7dd;
        border-radius: 8px;
        background: #fffdf6;
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
        background: #f8f6ef;
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
        max-height: min(520px, 42dvh);
        overflow: auto;
        border-bottom: 1px solid #d7d2c8;
        padding-bottom: 12px;
    }

    .console-strip {
        display: grid;
        align-content: center;
        gap: 6px;
        background: #070707;
        color: #d7d2c8;
        padding: 10px 12px;
        font-size: 11px;
    }

    .console-strip strong {
        color: #f8f6ef;
    }

    @media (max-width: 1100px) {
        .main-grid {
            grid-template-columns: 1fr;
        }

        .graph-workspace {
            grid-template-columns: 1fr;
            grid-template-rows: minmax(420px, 1fr) auto;
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
            padding: 10px;
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
    }
</style>
