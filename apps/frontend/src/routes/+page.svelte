<script lang="ts">
    import { onMount } from "svelte";
    import type {
        Artifact,
        ChatQueryResult,
        CodexPlugin,
        FindingsResult,
        GraphData,
        GraphEntity,
        ServiceStatus,
        WorkspaceStatus,
    } from "$lib/types";
    import {
        API_URL,
        apiFetch,
        cleanupLiveEvidence,
        deleteWorkspace,
        getArtifacts,
        getGraphData,
        getWorkspaceStatus,
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
    } from "$lib/workspace/projectStore";
    import ChatPanel from "$lib/components/chat/ChatPanel.svelte";
    import ConfirmModal from "$lib/components/ConfirmModal.svelte";
    import ActivityView from "$lib/components/insights/ActivityView.svelte";
    import FindingsView from "$lib/components/insights/FindingsView.svelte";
    import GraphView from "$lib/components/insights/GraphView.svelte";
    import WorkspaceSummary from "$lib/components/insights/WorkspaceSummary.svelte";
    import {
        demoArtifacts,
        demoChatQueryResult,
        demoFindings,
        demoGraphData,
        demoWorkspaceStatus,
    } from "$lib/chat/demoWorkspace";
    import { buildGraphLinks } from "$lib/graph/viewModel";
    import { runAnalysis } from "$lib/findings/analysisRunner";
    import {
        assistantMsg,
        classifyChatCommand,
        runChatQuery,
        userMsg,
    } from "$lib/chat/controller";

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
    let lastChatResult: ChatQueryResult | null = null;
    let lastFindings: FindingsResult | null = null;
    let lastAnalysisAt = "";
    let activityArtifacts: Artifact[] = [];
    let removeConfirmOpen = false;
    let workspacePendingRemoval = "";
    let removeInProgress = false;
    let walkthroughOpen = false;
    let paneSplit = 52;
    let mainGrid: HTMLElement | null = null;
    let resizingPanes = false;
    let workspaceRefreshRunID = 0;

    $: readySources = $project.connectors.filter(
        (source) => source.status === "ready",
    );
    $: graphEntities = graphData?.entities ?? [];
    $: graphRelationships = graphData?.relationships ?? [];
    $: graphLinks = buildGraphLinks(graphEntities, graphRelationships);
    $: mismatchCount =
        workspaceStatus?.mismatch_count ?? lastFindings?.mismatch_count ?? 0;
    $: statusLine = buildStatusLine(
        apiStatus,
        workerStatus,
        codexInstalled,
        codexLoggedIn,
        codexAccount,
    );
    $: codexLabel = codexLoggedIn
        ? normalizeCodexAccount(codexAccount)
        : "Codex not logged in";
    $: sourceSummary = `${readySources.length} source${readySources.length === 1 ? "" : "s"}`;
    $: topContext = `${$project.name} · ${sourceSummary}`;
    $: hasSources = readySources.length > 0;
    $: recentArtifacts =
        activityArtifacts.length > 0
            ? activityArtifacts
            : (lastChatResult?.artifacts ?? []);
    $: protectedWorkspace =
        workspacePath === DEFAULT_WORKSPACE_PATH ||
        workspacePath === DEMO_WORKSPACE_PATH;

    onMount(() => {
        const savedPath = localStorage.getItem("contextos_workspace_path");
        workspacePath = savedPath || getProject().workspacePath;
        openProject(workspacePath);
        void refreshSystemStatus();
        void hydrateWorkspaces();
        void refreshWorkspace();
    });

    async function refreshSystemStatus() {
        const apiProbe = probeService(API_URL).then((status) => {
            apiStatus = status;
            return status;
        });
        const workerProbe = probeService("/worker").then((status) => {
            workerStatus = status;
            return status;
        });

        const currentAPIStatus = await apiProbe;
        if (currentAPIStatus === "ok") {
            await checkCodexStatus();
        } else {
            setCodexUnavailable();
        }
        await workerProbe.catch(() => {
            workerStatus = "unreachable";
        });
    }

    async function checkCodexStatus() {
        try {
            const res = await apiFetch(`${API_URL}/codex/status`, {
                signal: AbortSignal.timeout(5000),
            });
            if (!res.ok) {
                setCodexUnavailable();
                return;
            }
            const body = await res.json();
            codexInstalled = body?.installed === true;
            codexLoggedIn = body?.logged_in === true;
            codexAccount = body?.account ?? "";
            codexPlugins = body?.plugins ?? [];
        } catch {
            setCodexUnavailable();
        }
    }

    function setCodexUnavailable() {
        codexInstalled = false;
        codexLoggedIn = false;
        codexAccount = "";
        codexPlugins = [];
    }

    async function refreshWorkspace() {
        const runID = ++workspaceRefreshRunID;
        if (workspacePath === DEMO_WORKSPACE_PATH) {
            workspaceStatus = demoWorkspaceStatus();
            graphData = demoGraphData();
            activityArtifacts = demoArtifacts();
            lastFindings = demoFindings();
            lastAnalysisAt = "2026-01-01T09:30:00.000Z";
            return;
        }
        const status = await loadWorkspaceStatus(workspacePath);
        if (runID !== workspaceRefreshRunID) return;
        workspaceStatus = status;

        const confirmedSources = getProject().connectors.filter(
            (source) => source.status === "ready",
        );
        if (confirmedSources.length === 0) {
            graphData = null;
            selectedEntity = null;
            activityArtifacts = [];
            return;
        }

        const [nextGraphData, artifacts] = await Promise.all([
            getGraphData(workspacePath),
            getArtifacts({
                workspace_id: workspacePath,
                limit: 100,
            }),
        ]);
        if (runID !== workspaceRefreshRunID) return;
        graphData = nextGraphData;
        activityArtifacts = artifacts?.artifacts ?? [];
    }

    async function cleanNoisyLiveEvidence() {
        if (workspacePath === DEMO_WORKSPACE_PATH) {
            return "Demo Activity is read-only.";
        }
        const result = await cleanupLiveEvidence(workspacePath);
        if (!result.ok) {
            throw new Error(
                result.body.message ??
                    result.body.error ??
                    "Activity cleanup failed.",
            );
        }
        await refreshWorkspace();
        return `Removed ${result.body.deleted_count} noisy live evidence event${result.body.deleted_count === 1 ? "" : "s"}.`;
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

    function startPaneResize(event: PointerEvent) {
        if (!mainGrid) return;
        resizingPanes = true;
        updatePaneSplit(event.clientX);
        window.addEventListener("pointermove", handlePaneResize);
        window.addEventListener("pointerup", stopPaneResize, { once: true });
        window.addEventListener("pointercancel", stopPaneResize, {
            once: true,
        });
    }

    function handlePaneResize(event: PointerEvent) {
        updatePaneSplit(event.clientX);
    }

    function stopPaneResize() {
        resizingPanes = false;
        window.removeEventListener("pointermove", handlePaneResize);
        window.removeEventListener("pointerup", stopPaneResize);
        window.removeEventListener("pointercancel", stopPaneResize);
    }

    function updatePaneSplit(clientX: number) {
        if (!mainGrid) return;
        const rect = mainGrid.getBoundingClientRect();
        const next = ((clientX - rect.left) / rect.width) * 100;
        paneSplit = Math.max(32, Math.min(68, Math.round(next)));
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

    async function submitCommand() {
        const text = command.trim();
        if (!text || busy) return;
        command = "";
        addMessage(userMsg(text));
        await routeCommand(text);
    }

    async function routeCommand(text: string) {
        const action = classifyChatCommand(text);
        if (action === "clear") {
            clearChat();
            lastChatResult = null;
            lastFindings = null;
            addMessage(
                assistantMsg("Chat history cleared for this workspace."),
            );
            return;
        }
        if (action === "openSources") {
            sourcePanelOpen = true;
            addMessage(
                assistantMsg("Source setup is open in the workspace panel."),
            );
            return;
        }
        if (action === "runFindings") {
            await runFindings();
            return;
        }
        await runChatQuery({
            text,
            workspacePath,
            addMessage,
            replaceMessage,
            setBusy: (value) => (busy = value),
            setLastChatResult: (result) => (lastChatResult = result),
            setActivityArtifacts: (artifacts) =>
                (activityArtifacts = artifacts),
            refreshWorkspace,
        });
    }

    async function runFindings() {
        await runAnalysis({
            workspacePath,
            readySources: getProject().connectors.filter(
                (source) => source.status === "ready",
            ),
            addMessage,
            replaceMessage,
            setBusy: (value) => (busy = value),
            setLastFindings: (result) => (lastFindings = result),
            setLastAnalysisAt: (value) => (lastAnalysisAt = value),
            openSources: () => (sourcePanelOpen = true),
            refreshWorkspace,
        });
    }

    async function handleKnowledgeDone() {
        sourcePanelOpen = false;
        await Promise.all([refreshWorkspace(), refreshSystemStatus()]);
        addMessage(assistantMsg("Source setup updated."));
    }

    async function handleKnowledgeReset() {
        sourcePanelOpen = false;
        lastChatResult = null;
        lastFindings = null;
        lastAnalysisAt = "";
        graphData = null;
        selectedEntity = null;
        activityArtifacts = [];
        workspaceStatus = null;
        activeInsightTab = "findings";
        await Promise.all([refreshWorkspace(), refreshSystemStatus()]);
        addMessage(
            assistantMsg(
                "All workspace data reset. Analysis results, graph data, activity, saved sources, chat history, and local workspace memory were cleared.",
            ),
        );
    }

    function switchInsightTab(tab: "findings" | "graph" | "activity") {
        activeInsightTab = tab;
        sourcePanelOpen = false;
        if (tab === "graph" || tab === "activity") {
            void refreshWorkspace();
        }
    }

    function openGuideTarget(
        target: "sources" | "findings" | "graph" | "activity" | "agent",
    ) {
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
        const codex =
            currentApiStatus === "ok"
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

    function normalizeCodexAccount(value: string) {
        const clean = value
            .split("\n")
            .map((line) => line.trim())
            .filter(
                (line) => line && !line.toLowerCase().startsWith("warning:"),
            )
            .slice(-1)[0];
        return clean || "Codex logged in";
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
                on:change={(event) =>
                    switchWorkspace(
                        (event.currentTarget as HTMLSelectElement).value,
                    )}
            >
                {#each $workspaces as workspace (workspace.workspacePath)}
                    <option value={workspace.workspacePath}
                        >{workspace.name}</option
                    >
                {/each}
            </select>
            <form
                class="new-workspace"
                on:submit|preventDefault={createWorkspace}
            >
                <input
                    bind:value={newWorkspacePath}
                    placeholder="New workspace path"
                />
                <button type="submit" disabled={newWorkspacePath.trim() === ""}
                    >New</button
                >
                <button
                    type="button"
                    on:click={requestRemoveActiveWorkspace}
                    disabled={busy || protectedWorkspace}
                    title={protectedWorkspace
                        ? "Default and demo workspaces cannot be removed"
                        : "Remove workspace"}>Remove</button
                >
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
                API {apiStatus === "ok"
                    ? "Ready"
                    : apiStatus === "checking"
                      ? "Checking"
                      : "Offline"}
            </span>
            <span
                class="status-chip"
                class:status-ok={workerStatus === "ok"}
                class:status-checking={workerStatus === "checking"}
                class:status-offline={workerStatus === "unreachable"}
                title="AI worker status"
            >
                Worker {workerStatus === "ok"
                    ? "Ready"
                    : workerStatus === "checking"
                      ? "Checking"
                      : "Offline"}
            </span>
            <span
                class="status-chip"
                class:status-ok={apiStatus === "ok" &&
                    codexInstalled &&
                    codexLoggedIn}
                class:status-checking={apiStatus === "checking"}
                class:status-offline={apiStatus !== "ok" ||
                    !codexInstalled ||
                    !codexLoggedIn}
                title="Codex status"
            >
                Codex {apiStatus !== "ok"
                    ? "Unavailable"
                    : !codexInstalled
                      ? "Missing"
                      : !codexLoggedIn
                        ? "Login needed"
                        : "Connected"}
            </span>
        </div>
    </header>

    <section
        class="main-grid"
        bind:this={mainGrid}
        class:resizing={resizingPanes}
        style={`--pane-split:${paneSplit}%;`}
    >
        <section class="chat-pane" aria-label="Report agent chat">
            <ChatPanel
                messages={$chatMessages}
                {hasSources}
                {busy}
                bind:command
                onClear={clearChat}
                onSubmit={submitCommand}
            />
        </section>

        <button
            type="button"
            class="pane-resizer"
            aria-label="Resize chat and insights panes"
            title="Drag to resize panes"
            on:pointerdown={startPaneResize}
        ></button>

        <section class="insight-pane" aria-label="Project insights">
            <WorkspaceSummary
                {codexLabel}
                workspaceName={$project.name}
                {readySources}
                {codexLoggedIn}
                {codexPlugins}
                {sourcePanelOpen}
                onClose={() => (sourcePanelOpen = false)}
                onDone={handleKnowledgeDone}
                onReset={handleKnowledgeReset}
            />

            <section class="insight-card">
                <div class="insight-head">
                    <nav aria-label="Insight tabs">
                        <button
                            type="button"
                            class:active={activeInsightTab === "findings"}
                            on:click={() => switchInsightTab("findings")}
                            >Findings</button
                        >
                        <button
                            type="button"
                            class:active={activeInsightTab === "graph"}
                            on:click={() => switchInsightTab("graph")}
                            >Graph</button
                        >
                        <button
                            type="button"
                            class:active={activeInsightTab === "activity"}
                            on:click={() => switchInsightTab("activity")}
                            >Activity</button
                        >
                    </nav>
                    <button
                        type="button"
                        on:click={runFindings}
                        disabled={!hasSources || busy}
                        >{busy ? "Running" : "Run Analysis"}</button
                    >
                </div>

                {#if activeInsightTab === "findings"}
                    <FindingsView
                        {lastFindings}
                        {lastAnalysisAt}
                        {recentArtifacts}
                        readySourceCount={readySources.length}
                        {workspaceStatus}
                        {hasSources}
                    />
                {:else if activeInsightTab === "graph"}
                    <GraphView
                        {graphData}
                        bind:selectedEntity
                        {hasSources}
                    />
                {:else}
                    <ActivityView
                        {recentArtifacts}
                        onCleanupNoisyLiveEvidence={cleanNoisyLiveEvidence}
                    />
                {/if}
            </section>
        </section>
    </section>

    <footer class="console-strip">
        <strong>{topContext}</strong>
        <span>{statusLine}</span>
        <span
            >{graphData?.entity_count ?? graphData?.count ?? 0} graph nodes | {graphData?.relationship_count ??
                graphLinks.length} links | {mismatchCount} findings</span
        >
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
        <ConfirmModal
            eyebrow="DELETE WORKSPACE"
            title="Remove this workspace?"
            description="This deletes database memory, analysis results, graph snapshots, parsed local JSON, chat, selected sources, and project state for "
            detail={workspacePendingRemoval}
            confirmLabel="Delete"
            busyLabel="Deleting"
            busy={removeInProgress}
            on:cancel={cancelWorkspaceRemoval}
            on:confirm={confirmWorkspaceRemoval}
        />
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
                <button
                    type="button"
                    on:click={() => openGuideTarget("sources")}
                >
                    <strong>Sources</strong>
                    <p>Connect or select source data.</p>
                </button>
                <button
                    type="button"
                    on:click={() => openGuideTarget("findings")}
                >
                    <strong>Findings</strong>
                    <p>Read issues, dates, evidence, and actions.</p>
                </button>
                <button type="button" on:click={() => openGuideTarget("graph")}>
                    <strong>Graph</strong>
                    <p>Inspect entities and relationships.</p>
                </button>
                <button
                    type="button"
                    on:click={() => openGuideTarget("activity")}
                >
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
    .insight-head,
    .console-strip {
        letter-spacing: 0.05em;
    }

    .topbar button,
    .new-workspace button,
    .insight-head button {
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background-color: transparent;
        background-image: linear-gradient(
            90deg,
            #1c1b18 0 50%,
            transparent 50% 100%
        );
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
    .insight-head nav button.active {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .topbar button:disabled,
    .new-workspace button:disabled,
    .insight-head button:disabled {
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
        grid-template-columns: minmax(360px, var(--pane-split, 52%)) 9px minmax(
                360px,
                1fr
            );
        border-bottom: 1px solid #d7d2c8;
    }

    .main-grid.resizing,
    .main-grid.resizing * {
        cursor: col-resize;
        user-select: none;
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
        padding: 16px;
    }

    .pane-resizer {
        width: 9px;
        min-width: 9px;
        height: 100%;
        border: 0;
        border-left: 1px solid #d7d2c8;
        border-right: 1px solid transparent;
        border-radius: 0;
        background: #ebe8e0;
        cursor: col-resize;
        padding: 0;
        position: relative;
    }

    .pane-resizer::before {
        content: "";
        position: absolute;
        top: 12px;
        bottom: 12px;
        left: 3px;
        border-left: 1px solid #bdb7a8;
        opacity: 0;
        transition: opacity 0.15s ease;
    }

    .pane-resizer:hover,
    .main-grid.resizing .pane-resizer {
        background: #f1eee5;
    }

    .pane-resizer:hover::before,
    .main-grid.resizing .pane-resizer::before {
        opacity: 1;
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

    .insight-head > button {
        padding: 8px 12px;
        white-space: nowrap;
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
        background-image: linear-gradient(
            90deg,
            #1c1b18 0 50%,
            transparent 50% 100%
        );
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

    @media (max-width: 1100px) {
        .main-grid {
            grid-template-columns: 1fr;
        }

        .pane-resizer {
            display: none;
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
    }
</style>
