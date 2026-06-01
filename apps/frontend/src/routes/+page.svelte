<script lang="ts">
    import { onMount } from "svelte";
    import type {
        ChatMessage,
        ChatQueryResult,
        CodexPlugin,
        ConnectorKind,
        FindingsResult,
        GraphData,
        GraphEntity,
        ServiceStatus,
        WorkspaceStatus,
    } from "$lib/types";
    import {
        API_URL,
        getGraphData,
        getWorkspaceStatus,
        postChatQuery,
        postFindings,
        probeService,
    } from "$lib/api";
    import {
        addMessage,
        addWorkspace,
        chatMessages,
        clearChat,
        getProject,
        hydrateWorkspaces,
        loadWorkspaceStatus,
        openProject,
        project,
        replaceMessage,
        workspaces,
    } from "$lib/projectStore";
    import KnowledgeInstall from "$lib/components/knowledge/KnowledgeInstall.svelte";

    type GithubStatus = "checking" | "connected" | "disconnected";
    type ViewMode = "graph" | "dual" | "workspace";

    let apiStatus: ServiceStatus = "checking";
    let githubStatus: GithubStatus = "checking";
    let githubLogin = "";
    let codexLoggedIn = false;
    let codexInstalled = false;
    let codexAccount = "";
    let codexPlugins: CodexPlugin[] = [];
    let workspacePath = "/workspace";
    let command = "";
    let busy = false;
    let sourcePanelOpen = false;
    let viewMode: ViewMode = "dual";
    let workspaceStatus: WorkspaceStatus | null = null;
    let graphData: GraphData | null = null;
    let lastChatResult: ChatQueryResult | null = null;
    let lastFindings: FindingsResult | null = null;

    $: readySources = $project.connectors.filter(
        (source) => source.status === "ready",
    );
    $: activeSources = $project.connectors.filter(
        (source) => source.status !== "idle",
    );
    $: graphEntities = graphData?.entities ?? [];
    $: selectedEntity = graphEntities[0] ?? null;
    $: eventCount = workspaceStatus?.event_count ?? 0;
    $: mismatchCount =
        workspaceStatus?.mismatch_count ?? lastFindings?.mismatch_count ?? 0;
    $: statusLine = buildStatusLine(
        apiStatus,
        codexInstalled,
        codexLoggedIn,
        codexAccount,
        githubStatus,
        githubLogin,
    );

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
        await Promise.all([checkCodexStatus(), checkGithubStatus()]);
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

    async function checkGithubStatus() {
        try {
            const res = await fetch(`${API_URL}/github/status`);
            if (!res.ok) {
                githubStatus = "disconnected";
                return;
            }
            const body = await res.json();
            githubStatus = body?.connected === true ? "connected" : "disconnected";
            githubLogin = body?.login ?? "";
        } catch {
            githubStatus = "disconnected";
        }
    }

    async function refreshWorkspace() {
        await loadWorkspaceStatus(workspacePath);
        workspaceStatus = await getWorkspaceStatus(workspacePath);
        graphData = await getGraphData(workspacePath);
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
                local_date: new Date().toISOString().slice(0, 10),
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

        const latest = ready[ready.length - 1];
        const load = loadingMsg(`Running local analysis for ${latest.connector}...`);
        addMessage(load);
        busy = true;
        try {
            const codexOnlyConnectors = new Set<ConnectorKind>([
                "slack",
                "notion",
                "sharepoint",
                "googledrive",
            ]);
            const provider = codexOnlyConnectors.has(latest.connector)
                ? "codex"
                : "token";
            const res = await postFindings({
                workspace_id: workspacePath,
                connector: latest.connector,
                uri: latest.uri,
                provider,
                role: "pmo",
                include_execution: false,
            });
            if (res.ok) {
                lastFindings = res.body;
                replaceMessage(
                    load.id,
                    assistantMsg(res.body.summary || "Analysis complete.", {
                        kind: "findings",
                        findingsResult: res.body,
                    }),
                );
                return;
            }
            replaceMessage(
                load.id,
                assistantMsg(
                    `Analysis failed: ${res.body.message ?? res.body.error ?? "unknown error"}`,
                ),
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

    async function switchWorkspace(path: string) {
        if (path === workspacePath) return;
        workspacePath = path;
        openProject(path);
        lastChatResult = null;
        lastFindings = null;
        await refreshWorkspace();
    }

    async function createWorkspace() {
        const path = prompt("Workspace folder path:", workspacePath);
        if (!path?.trim()) return;
        addWorkspace(path.trim());
        workspacePath = path.trim();
        await refreshWorkspace();
    }

    async function handleKnowledgeDone() {
        sourcePanelOpen = false;
        await Promise.all([refreshWorkspace(), refreshSystemStatus()]);
        addMessage(assistantMsg("Source setup updated."));
    }

    function buildStatusLine(
        currentApiStatus: ServiceStatus,
        currentCodexInstalled: boolean,
        currentCodexLoggedIn: boolean,
        currentCodexAccount: string,
        currentGithubStatus: GithubStatus,
        currentGithubLogin: string,
    ) {
        if (currentApiStatus === "checking") return "Checking API";
        if (currentApiStatus !== "ok") return "API offline";
        if (!currentCodexInstalled) return "Codex CLI not installed";
        if (!currentCodexLoggedIn) return "Codex CLI not logged in";
        if (currentGithubStatus === "connected") {
            return `Codex connected; GitHub connected${currentGithubLogin ? ` as ${currentGithubLogin}` : ""}`;
        }
        return `Codex connected${currentCodexAccount ? ` as ${currentCodexAccount}` : ""}`;
    }

    function nodeStyle(index: number) {
        const col = index % 18;
        const row = Math.floor(index / 18);
        const x = 8 + col * 5 + (row % 2) * 1.8;
        const y = 12 + (row % 11) * 7;
        return `left:${Math.min(x, 94)}%;top:${Math.min(y, 90)}%;`;
    }

    function edgeStyle(index: number) {
        const row = Math.floor(index / 14);
        const x = 4 + (index % 14) * 6.8;
        const y = 10 + (row % 9) * 8.5;
        const rotate = -24 + (index % 12) * 4;
        const width = 72 + (index % 5) * 34;
        return `left:${x}%;top:${y}%;width:${width}px;transform:rotate(${rotate}deg);`;
    }

    function entityClass(entity: GraphEntity) {
        const value = entity.type.toLowerCase();
        if (value.includes("person")) return "person";
        if (value.includes("org") || value.includes("company")) return "org";
        if (value.includes("feature") || value.includes("project")) return "feature";
        return "entity";
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
</script>

<svelte:head>
    <title>ContextOS</title>
</svelte:head>

<main class="app-shell" class:graph-only={viewMode === "graph"} class:workspace-only={viewMode === "workspace"}>
    <header class="topbar">
        <strong>CONTEXTOS</strong>
        <nav aria-label="Workspace view mode">
            <button class:active={viewMode === "graph"} on:click={() => (viewMode = "graph")}>Graph</button>
            <button class:active={viewMode === "dual"} on:click={() => (viewMode = "dual")}>Dual</button>
            <button class:active={viewMode === "workspace"} on:click={() => (viewMode = "workspace")}>Workspace</button>
        </nav>
        <div class="top-status">
            <span>Step 2/5</span>
            <strong>{apiStatus === "ok" ? "Ready" : "Offline"}</strong>
        </div>
    </header>

    <section class="main-grid">
        {#if viewMode !== "workspace"}
            <section class="graph-pane" aria-label="Graph relationship visualization">
                <div class="pane-title">
                    <span>Graph Relationship Visualization</span>
                    <div>
                        <button on:click={refreshWorkspace}>Refresh</button>
                        <button on:click={() => (viewMode = "dual")}>Focus</button>
                    </div>
                </div>

                <div class="graph-canvas">
                    {#each graphEntities.slice(0, 140) as entity, index (entity.id)}
                        <span class="edge" style={edgeStyle(index)}></span>
                        <button class={`node ${entityClass(entity)}`} style={nodeStyle(index)} title={entity.name}>
                            <span></span>
                            <em>{entity.name}</em>
                        </button>
                    {/each}

                    {#if graphEntities.length === 0}
                        <div class="empty-graph">
                            <strong>No graph data yet</strong>
                            <p>Ingest a source or run analysis to populate local entities.</p>
                        </div>
                    {/if}

                    {#if selectedEntity}
                        <aside class="node-card">
                            <div>
                                <span>Node Details</span>
                                <strong>{selectedEntity.type}</strong>
                            </div>
                            <p><b>Name:</b> {selectedEntity.name}</p>
                            <p><b>UUID:</b> {selectedEntity.id}</p>
                            <p><b>Confidence:</b> {Math.round((selectedEntity.confidence ?? 0) * 100)}%</p>
                            <hr />
                            <p>{selectedEntity.evidence?.[0] ?? "Evidence appears after source ingestion and analysis."}</p>
                        </aside>
                    {/if}

                    <div class="legend">
                        <strong>ENTITY TYPES</strong>
                        <span><i class="entity"></i>Entity</span>
                        <span><i class="org"></i>Organization</span>
                        <span><i class="person"></i>Person</span>
                        <span><i class="feature"></i>Feature</span>
                    </div>
                </div>
            </section>
        {/if}

        {#if viewMode !== "graph"}
            <section class="work-pane" aria-label="Workspace operations">
                <section class="stage-card">
                    <div class="stage-head">
                        <div>
                            <p>Workspace</p>
                            <h1>{$project.name}</h1>
                        </div>
                        <button on:click={refreshSystemStatus}>Refresh status</button>
                    </div>
                    <p class="muted">{statusLine}</p>

                    <div class="metrics">
                        <div><strong>{eventCount}</strong><span>local events</span></div>
                        <div><strong>{readySources.length}</strong><span>ready sources</span></div>
                        <div><strong>{graphData?.count ?? 0}</strong><span>graph nodes</span></div>
                        <div><strong>{mismatchCount}</strong><span>findings</span></div>
                    </div>
                </section>

                <section class="stage-list">
                    <div><span>PL</span><strong>Planning / outline</strong><em>Ready</em></div>
                    <div><span>01</span><strong>Workspace selected</strong><em>{workspacePath}</em></div>
                    <div><span>02</span><strong>Sources connected</strong><em>{activeSources.length}</em></div>
                    <div><span>03</span><strong>Local chat query</strong><em>{lastChatResult?.intent ?? "waiting"}</em></div>
                    <div><span>OK</span><strong>Truth panel</strong><em>{mismatchCount} findings</em></div>
                </section>

                <section class="tool-grid">
                    <button on:click={() => (sourcePanelOpen = true)}><strong>Source Setup</strong><span>Add or repair connectors</span></button>
                    <button on:click={runFindings}><strong>Run Analysis</strong><span>Generate local findings</span></button>
                    <button on:click={createWorkspace}><strong>Workspace</strong><span>Add or switch context</span></button>
                    <button on:click={() => routeCommand("status")}><strong>Status</strong><span>Ask local status</span></button>
                </section>

                <section class="workspace-row">
                    {#each $workspaces as workspaceItem (workspaceItem.workspacePath)}
                        <button class:active={workspaceItem.workspacePath === workspacePath} on:click={() => switchWorkspace(workspaceItem.workspacePath)}>
                            <strong>{workspaceItem.name}</strong>
                            <small>{workspaceItem.workspacePath}</small>
                        </button>
                    {/each}
                </section>

                <section class="chat-card">
                    <div class="chat-head">
                        <div>
                            <strong>Report Agent - Chat</strong>
                            <span>Local source answer and evidence</span>
                        </div>
                        <button on:click={() => clearChat()}>Clear</button>
                    </div>

                    <div class="messages" aria-live="polite">
                        {#if $chatMessages.length === 0}
                            <article class="message assistant">
                                <span>CTX</span>
                                <p>Ask for Slack messages, GitHub PRs, Jira tickets, docs, findings, or workspace status. Answers stay local to this workspace.</p>
                            </article>
                        {:else}
                            {#each $chatMessages as message (message.id)}
                                <article class="message" class:user={message.role === "user"}>
                                    <span>{message.role === "user" ? "YOU" : "CTX"}</span>
                                    <p>{message.loading ? "Working..." : message.text}</p>
                                    {#if message.card?.chatResult?.artifacts?.length}
                                        <details>
                                            <summary>{message.card.chatResult.artifact_count} evidence items</summary>
                                            {#each message.card.chatResult.artifacts.slice(0, 5) as artifact (artifact.id)}
                                                <div class="evidence-item">
                                                    <strong>{artifact.title || artifact.source_uri}</strong>
                                                    <small>{artifact.connector} | {formatTime(artifact.ingested_at)}</small>
                                                    <p>{artifact.preview}</p>
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
                            disabled={busy}
                            placeholder="Ask: give me today Slack messages, recent GitHub PRs, show findings..."
                        />
                        <button disabled={busy || command.trim() === ""}>Send</button>
                    </form>
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
                {/if}
            </section>
        {/if}
    </section>

    <footer class="console-strip">
        <strong>SYSTEM DASHBOARD</strong>
        <span>{new Date().toLocaleTimeString()} | {statusLine}</span>
        <span>{eventCount} events | {graphData?.count ?? 0} graph nodes | {mismatchCount} findings</span>
    </footer>
</main>

<style>
    :global(body) {
        margin: 0;
        background: #ebe8e0;
        color: #1c1b18;
        font-family: "IBM Plex Sans", "Aptos", "Segoe UI", sans-serif;
    }

    :global(*) {
        box-sizing: border-box;
    }

    button,
    input {
        font: inherit;
    }

    button {
        cursor: pointer;
    }

    .app-shell {
        min-height: 100vh;
        display: grid;
        grid-template-rows: 40px minmax(0, 1fr) 74px;
        background: #ebe8e0;
    }

    .topbar {
        display: grid;
        grid-template-columns: 1fr auto 1fr;
        align-items: center;
        border-bottom: 1px solid #d7d2c8;
        background: #f8f6ef;
        padding: 0 12px;
    }

    .topbar strong,
    .topbar button,
    .top-status,
    .pane-title,
    .stage-list,
    .tool-grid,
    .chat-head span,
    .message span,
    .console-strip {
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        letter-spacing: 0.05em;
    }

    .topbar nav {
        display: flex;
        gap: 4px;
        border-radius: 8px;
        background: #ebe8e0;
        padding: 4px;
    }

    .topbar button,
    .pane-title button,
    .stage-head button,
    .tool-grid button,
    .workspace-row button,
    .chat-head button,
    .composer button {
        border: 1px solid #d7d2c8;
        border-radius: 6px;
        background: #f8f6ef;
        color: #1c1b18;
    }

    .topbar button {
        min-width: 76px;
        padding: 5px 12px;
        font-size: 12px;
    }

    .topbar button.active {
        background: #1c1b18;
        color: #f8f6ef;
    }

    .top-status {
        display: flex;
        justify-content: flex-end;
        gap: 16px;
        color: #8a8678;
        font-size: 12px;
    }

    .top-status strong {
        color: #2d6a4f;
    }

    .main-grid {
        min-height: 0;
        display: grid;
        grid-template-columns: minmax(480px, 1fr) minmax(460px, 0.92fr);
        border-bottom: 1px solid #d7d2c8;
    }

    .graph-only .main-grid,
    .workspace-only .main-grid {
        grid-template-columns: 1fr;
    }

    .graph-pane,
    .work-pane {
        min-width: 0;
        min-height: 0;
        overflow: hidden;
    }

    .graph-pane {
        display: grid;
        grid-template-rows: 36px minmax(0, 1fr);
        border-right: 1px solid #d7d2c8;
        background: #f8f6ef;
    }

    .pane-title {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 10px;
        color: #1c1b18;
        font-size: 12px;
        font-weight: 700;
    }

    .pane-title div {
        display: flex;
        gap: 6px;
    }

    .pane-title button {
        padding: 5px 9px;
        font-size: 12px;
    }

    .graph-canvas {
        position: relative;
        overflow: hidden;
        background:
            radial-gradient(circle, rgba(28, 27, 24, 0.12) 1px, transparent 1px) 0 0 / 18px 18px,
            #fbfaf5;
    }

    .edge {
        position: absolute;
        height: 1px;
        background: rgba(138, 134, 120, 0.28);
        transform-origin: left center;
    }

    .node {
        position: absolute;
        display: inline-flex;
        align-items: center;
        gap: 4px;
        border: 0;
        background: transparent;
        color: #535047;
        padding: 0;
        transform: translate(-50%, -50%);
    }

    .node span,
    .legend i {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: #1f5f8b;
        display: inline-block;
        flex: 0 0 auto;
    }

    .node em {
        max-width: 92px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
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

    .node-card,
    .legend,
    .empty-graph {
        position: absolute;
        border: 1px solid #d7d2c8;
        border-radius: 12px;
        background: rgba(248, 246, 239, 0.96);
        box-shadow: 0 14px 36px rgba(28, 27, 24, 0.08);
    }

    .node-card {
        top: 54px;
        right: 18px;
        width: 292px;
        padding: 16px;
        font-size: 13px;
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

    .legend {
        left: 18px;
        bottom: 18px;
        display: grid;
        grid-template-columns: repeat(2, auto);
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
    }

    .empty-graph {
        left: 50%;
        top: 50%;
        transform: translate(-50%, -50%);
        padding: 18px;
        text-align: center;
    }

    .work-pane {
        display: flex;
        flex-direction: column;
        gap: 14px;
        overflow: auto;
        padding: 16px;
    }

    .stage-card,
    .stage-list div,
    .tool-grid button,
    .workspace-row button,
    .chat-card,
    .setup-panel {
        border: 1px solid #d7d2c8;
        border-radius: 10px;
        background: #f8f6ef;
    }

    .stage-card {
        padding: 18px;
    }

    .stage-head {
        display: flex;
        justify-content: space-between;
        gap: 16px;
        align-items: start;
    }

    .stage-head p,
    .kicker {
        margin: 0;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        color: #8a8678;
        text-transform: uppercase;
        font-size: 12px;
    }

    .stage-head h1 {
        margin: 4px 0 0;
        font-size: clamp(30px, 4vw, 52px);
        line-height: 0.95;
        overflow-wrap: anywhere;
    }

    .stage-head button {
        padding: 8px 12px;
    }

    .muted,
    small {
        color: #8a8678;
    }

    .metrics {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 10px;
        margin-top: 18px;
    }

    .metrics div {
        border: 1px solid #ece7dd;
        border-radius: 8px;
        padding: 12px;
        text-align: center;
    }

    .metrics strong,
    .metrics span {
        display: block;
    }

    .metrics strong {
        font-size: 24px;
    }

    .metrics span {
        color: #8a8678;
        font-size: 12px;
    }

    .stage-list {
        display: grid;
        gap: 8px;
        font-size: 12px;
    }

    .stage-list div {
        min-height: 44px;
        display: grid;
        grid-template-columns: 34px 1fr auto;
        gap: 10px;
        align-items: center;
        padding: 0 12px;
    }

    .stage-list span {
        color: #2d6a4f;
    }

    .stage-list em {
        max-width: 170px;
        color: #8a8678;
        font-style: normal;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .tool-grid {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        gap: 10px;
    }

    .tool-grid button {
        min-height: 78px;
        padding: 12px;
        text-align: left;
    }

    .tool-grid strong,
    .tool-grid span {
        display: block;
    }

    .tool-grid span {
        margin-top: 6px;
        color: #8a8678;
        font-size: 12px;
    }

    .workspace-row {
        display: flex;
        gap: 8px;
        overflow-x: auto;
        padding-bottom: 2px;
    }

    .workspace-row button {
        min-width: 172px;
        padding: 10px;
        text-align: left;
    }

    .workspace-row button.active {
        border-color: #1c1b18;
    }

    .workspace-row strong,
    .workspace-row small {
        display: block;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .chat-card {
        min-height: 420px;
        display: grid;
        grid-template-rows: auto minmax(240px, 1fr) auto;
        overflow: hidden;
    }

    .chat-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
        border-bottom: 1px solid #d7d2c8;
        padding: 14px 16px;
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
        display: flex;
        flex-direction: column;
        gap: 12px;
        overflow: auto;
        padding: 16px;
    }

    .message {
        width: min(680px, 90%);
        border-radius: 14px;
        background: #ebe8e0;
        padding: 14px;
        line-height: 1.5;
    }

    .message.user {
        align-self: flex-end;
        background: #1c1b18;
        color: #f8f6ef;
    }

    .message span {
        display: block;
        margin-bottom: 6px;
        color: #8a8678;
        font-size: 12px;
    }

    .message p {
        margin: 0;
        white-space: pre-wrap;
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
        border-top: 1px solid #d7d2c8;
        padding-top: 10px;
    }

    .evidence-item strong,
    .evidence-item small {
        display: block;
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
        border: 1px solid #d7d2c8;
        border-radius: 8px;
        background: #fbfaf5;
        padding: 11px 12px;
        outline: none;
    }

    .composer button {
        padding: 0 18px;
        background: #1c1b18;
        color: #f8f6ef;
    }

    .setup-panel {
        max-height: 520px;
        overflow: auto;
    }

    .console-strip {
        display: grid;
        align-content: start;
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

        .graph-pane {
            min-height: 520px;
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

        .top-status {
            justify-content: flex-start;
        }

        .metrics,
        .tool-grid {
            grid-template-columns: repeat(2, 1fr);
        }

        .node-card {
            position: static;
            width: auto;
            margin: 12px;
        }
    }
</style>
