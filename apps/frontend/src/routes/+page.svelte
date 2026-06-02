<script lang="ts">
    import { onMount } from "svelte";
    import type {
        ChatMessage,
        ChatQueryResult,
        CodexPlugin,
        ConnectorKnowledge,
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
        chatMessages,
        clearChat,
        getProject,
        hydrateWorkspaces,
        loadWorkspaceStatus,
        openProject,
        project,
        replaceMessage,
    } from "$lib/projectStore";
    import KnowledgeInstall from "$lib/components/knowledge/KnowledgeInstall.svelte";

    type GithubStatus = "checking" | "connected" | "disconnected";

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
    let activeInsightTab: "findings" | "graph" | "activity" = "findings";
    let workspaceStatus: WorkspaceStatus | null = null;
    let graphData: GraphData | null = null;
    let selectedEntity: GraphEntity | null = null;
    let lastChatResult: ChatQueryResult | null = null;
    let lastFindings: FindingsResult | null = null;

    $: readySources = $project.connectors.filter(
        (source) => source.status === "ready",
    );
    $: graphEntities = graphData?.entities ?? [];
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
        githubStatus,
        githubLogin,
    );
    $: profileLabel = githubLogin || codexAccount || "profile";
    $: sourceSummary = `${readySources.length} source${readySources.length === 1 ? "" : "s"}`;
    $: topContext = `${profileLabel} · ${$project.name} · ${sourceSummary}`;
    $: hasSources = readySources.length > 0;
    $: sourceGroups = groupSources(readySources);
    $: recentArtifacts = lastChatResult?.artifacts ?? [];

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
        if (tab === "graph") {
            void refreshWorkspace();
        }
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
</script>

<svelte:head>
    <title>ContextOS</title>
</svelte:head>

<main class="app-shell">
    <header class="topbar">
        <strong>CONTEXTOS</strong>
        <div class="context-switcher" title={workspacePath}>{topContext}</div>
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
                            <span>CTX</span>
                            <p>{hasSources ? "Ask about Slack messages, GitHub PRs, Jira tickets, docs, findings, or recent activity." : "Connect GitHub repos, Slack channels, or docs first. After setup, chat will answer from those selected sources."}</p>
                        </article>
                    {:else}
                        {#each $chatMessages as message (message.id)}
                            <article class="message" class:user={message.role === "user"}>
                                <span>{message.role === "user" ? "YOU" : "CTX"}</span>
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
                                            <div class="evidence-item">
                                                <strong>{artifact.title || artifact.source_uri}</strong>
                                                <small>{artifact.connector} | {formatTime(artifact.ingested_at)}</small>
                                                <p>{previewText(artifact.preview)}</p>
                                                {#if (artifact.preview ?? "").length > 360}
                                                    <details class="full-evidence">
                                                        <summary>Full source text</summary>
                                                        <pre>{artifact.preview}</pre>
                                                    </details>
                                                {/if}
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
                    <button disabled={busy || !hasSources || command.trim() === ""}>Send</button>
                </form>
            </section>
        </section>

        <section class="insight-pane" aria-label="Project insights">
            <section class="source-strip">
                <div>
                    <span>PROFILE</span>
                    <strong>{profileLabel}</strong>
                </div>
                <div>
                    <span>PROJECT</span>
                    <strong>{$project.name}</strong>
                </div>
                <div>
                    <span>SOURCES</span>
                    <strong>{readySources.length}</strong>
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
                    <div class="graph-canvas">
                        {#each graphEntities.slice(0, 140) as entity, index (entity.id)}
                            <span class="edge" style={edgeStyle(index)}></span>
                            <button
                                type="button"
                                class={`node ${entityClass(entity)}`}
                                class:selected={selectedEntity?.id === entity.id}
                                style={nodeStyle(index)}
                                title={entity.name}
                                on:click={() => (selectedEntity = entity)}
                            >
                                <span></span>
                                <em>{entity.name}</em>
                            </button>
                        {/each}

                        {#if graphEntities.length === 0}
                            <div class="empty-graph">
                                <strong>No graph data yet</strong>
                                <p>{hasSources ? "Run analysis to populate local entities and relationships." : "Connect sources first, then run analysis to build the graph."}</p>
                            </div>
                        {/if}

                        {#if selectedEntity}
                            <aside class="node-card">
                                <div>
                                    <span>Node Details</span>
                                    <strong>{selectedEntity.type}</strong>
                                </div>
                                <p><b>Name:</b> {selectedEntity.name}</p>
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
                {:else}
                    <div class="activity-view">
                        {#if recentArtifacts.length}
                            {#each recentArtifacts.slice(0, 8) as artifact (artifact.id)}
                                <article>
                                    <span>{artifact.connector}</span>
                                    <strong>{artifact.title || artifact.source_uri}</strong>
                                    <p>{previewText(artifact.preview)}</p>
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
        <span>{graphData?.count ?? 0} graph nodes | {mismatchCount} findings</span>
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
    input {
        font: inherit;
    }

    button {
        cursor: pointer;
    }

    .app-shell {
        height: 100dvh;
        overflow: hidden;
        display: grid;
        grid-template-rows: 40px minmax(0, 1fr) 58px;
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
    .insight-head button,
    .chat-head button,
    .composer button {
        border: 1px solid #d7d2c8;
        border-radius: 6px;
        background: #f8f6ef;
        color: #1c1b18;
    }

    .context-switcher {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: #28261f;
        font-size: 12px;
        font-weight: 700;
        text-align: center;
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
        gap: 12px;
        overflow: auto;
        padding: 16px;
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

    .node:hover,
    .node.selected {
        color: #1c1b18;
        z-index: 3;
    }

    .node.selected em {
        font-weight: 700;
        text-decoration: underline;
        text-underline-offset: 3px;
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

    .source-strip,
    .source-summary,
    .insight-card,
    .sample-sources,
    .chat-card,
    .setup-panel {
        border: 1px solid #d7d2c8;
        border-radius: 10px;
        background: #f8f6ef;
    }

    .source-strip {
        display: grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: 10px;
        align-items: center;
        padding: 12px;
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

    .insight-head > button {
        padding: 8px 12px;
        white-space: nowrap;
    }

    .source-summary {
        display: flex;
        gap: 8px;
        overflow-x: auto;
        padding: 10px;
    }

    .source-summary div {
        min-width: 130px;
        border: 1px solid #ece7dd;
        border-radius: 8px;
        background: #fffdf7;
        padding: 8px 10px;
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
        padding: 12px;
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
    }

    .insight-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        border-bottom: 1px solid #d7d2c8;
        padding: 10px;
    }

    .insight-head nav {
        display: flex;
        gap: 4px;
        border-radius: 8px;
        background: #ebe8e0;
        padding: 4px;
    }

    .insight-head nav button {
        min-width: 86px;
        padding: 6px 10px;
        font-size: 12px;
    }

    .insight-head nav button.active {
        background: #1c1b18;
        color: #f8f6ef;
    }

    .findings-view,
    .activity-view {
        min-height: 0;
        display: flex;
        flex-direction: column;
        gap: 10px;
        overflow: auto;
        padding: 14px;
    }

    .findings-view article,
    .activity-view article,
    .empty-state {
        border: 1px solid #ece7dd;
        border-radius: 8px;
        background: #fffdf7;
        padding: 12px;
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
        border-top: 1px solid #d7d2c8;
        padding-top: 10px;
    }

    .evidence-item strong,
    .evidence-item small {
        display: block;
    }

    .evidence-item p {
        margin-top: 6px;
        color: #5f5b50;
    }

    .full-evidence {
        margin-top: 8px;
        border-top: 0;
        padding-top: 0;
    }

    .full-evidence pre {
        max-height: 260px;
        overflow: auto;
        white-space: pre-wrap;
        background: #fffdf7;
        border: 1px solid #ece7dd;
        border-radius: 8px;
        padding: 10px;
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
        max-height: min(520px, 42dvh);
        overflow: auto;
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

        .top-status {
            justify-content: flex-start;
        }

        .node-card {
            position: static;
            width: auto;
            margin: 12px;
        }
    }
</style>
