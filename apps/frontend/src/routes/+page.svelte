<script lang="ts">
  import { onMount } from "svelte";
  import { get } from "svelte/store";
  import type {
    ChatMessage,
    CodexPlugin,
    ConnectorKind,
    ServiceStatus,
  } from "$lib/types";
  import {
    API_URL,
    probeService,
    postFindings,
    streamCodexLogin,
  } from "$lib/api";
  import {
    addMessage,
    chatMessages,
    clearChat,
    getProject,
    markKnowledgeInstalled,
    openProject,
    project,
    replaceMessage,
  } from "$lib/projectStore";
  import ChatThread from "$lib/components/chat/ChatThread.svelte";
  import ChatInput from "$lib/components/chat/ChatInput.svelte";
  import KnowledgeInstall from "$lib/components/knowledge/KnowledgeInstall.svelte";

  // ---- service status ----
  let apiStatus: ServiceStatus = "checking";
  let codexLoggedIn = false;
  let codexPlugins: CodexPlugin[] = [];
  let codexInstalled = false;
  let codexAccount = "";
  let codexVersion = "";

  // ---- UI state ----
  let busy = false;
  let showKnowledge = false;
  let sidebarOpen = false;

  // ---- project path (workspace) ----
  let workspacePath = "/workspace";

  onMount(async () => {
    // Restore workspace path from localStorage if user set it before.
    const savedPath = localStorage.getItem("contextos_workspace_path");
    if (savedPath) workspacePath = savedPath;
    openProject(workspacePath);

    apiStatus = await probeService(API_URL);
    await checkCodexStatus();

    // Show knowledge install on first visit with no knowledge installed.
    if (!$project.knowledgeInstalledAt) {
      showKnowledge = true;
    }
  });

  async function checkCodexStatus() {
    try {
      const res = await fetch(`${API_URL}/codex/status`);
      if (res.ok) {
        const body = await res.json();
        codexInstalled = body?.installed === true;
        codexVersion = body?.version ?? "";
        codexLoggedIn = body?.logged_in === true;
        codexAccount = body?.account ?? "";
        codexPlugins = body?.plugins ?? [];
      }
    } catch {
      /* ignore */
    }
  }

  function changeWorkspace() {
    const p = prompt("Enter workspace folder path:", workspacePath);
    if (p && p.trim()) {
      workspacePath = p.trim();
      localStorage.setItem("contextos_workspace_path", workspacePath);
      openProject(workspacePath);
    }
  }

  // ---- chat command routing ----

  function makeId() {
    return Math.random().toString(36).slice(2) + Date.now().toString(36);
  }

  function now() {
    return new Date().toISOString();
  }

  function userMsg(text: string): ChatMessage {
    return { id: makeId(), role: "user", text, createdAt: now() };
  }

  function loadingMsg(): ChatMessage {
    return {
      id: makeId(),
      role: "assistant",
      text: "",
      createdAt: now(),
      loading: true,
    };
  }

  function assistantMsg(text: string, card?: ChatMessage["card"]): ChatMessage {
    return { id: makeId(), role: "assistant", text, createdAt: now(), card };
  }

  const HELP_TEXT = `Available commands:
• show findings [for <connector>] — Run analysis and show mismatches
• status — Show connector and service readiness
• install knowledge — Open the knowledge installation wizard
• clear — Clear chat history
• connectors — Open the full connector debug page

Or ask anything in natural language and I'll try to find relevant findings.`;

  async function handleSend(e: CustomEvent<string>) {
    const text = e.detail.trim();
    if (!text) return;

    addMessage(userMsg(text));
    await route(text.toLowerCase(), text);
  }

  async function route(lower: string, original: string) {
    if (lower === "help" || lower === "?") {
      addMessage(assistantMsg(HELP_TEXT));
      return;
    }

    if (lower === "clear") {
      clearChat();
      addMessage(assistantMsg("Chat cleared."));
      return;
    }

    if (lower === "connectors") {
      addMessage(assistantMsg("Opening connector debug page…"));
      window.location.href = "/connectors";
      return;
    }

    if (
      lower === "install knowledge" ||
      lower === "setup" ||
      lower === "install"
    ) {
      showKnowledge = true;
      addMessage(assistantMsg("Opening the knowledge installation wizard…"));
      return;
    }

    if (
      lower === "status" ||
      lower === "check status" ||
      lower === "are connectors ready"
    ) {
      await handleStatusCommand();
      return;
    }

    if (
      lower.startsWith("show findings") ||
      lower.startsWith("findings") ||
      lower.startsWith("analyse") ||
      lower.startsWith("analyze") ||
      lower.startsWith("mismatches") ||
      lower.startsWith("what are")
    ) {
      await handleFindingsCommand();
      return;
    }

    // Default: try to run findings for the most recent connector.
    await handleFindingsCommand();
  }

  async function handleStatusCommand() {
    const p = getProject();
    const statusMap: Record<string, boolean> = {};

    statusMap["API"] = apiStatus === "ok";
    statusMap["Codex CLI"] = codexInstalled;
    statusMap["Codex login"] = codexLoggedIn;

    const requiredPlugins = [
      "github@openai-curated",
      "atlassian-rovo@openai-curated",
      "slack@openai-curated",
      "google-drive@openai-curated",
      "notion@openai-curated",
      "sharepoint@openai-curated",
    ];
    for (const pl of requiredPlugins) {
      const short = pl.split("@")[0];
      statusMap[`${short} plugin`] = codexPlugins.some(
        (x) => x.name === pl && x.installed,
      );
    }

    for (const ck of p.connectors) {
      statusMap[`${ck.connector} knowledge`] = ck.status === "ready";
    }

    addMessage(
      assistantMsg(
        p.connectors.length === 0
          ? "No connectors ingested yet. Use 'install knowledge' to set them up."
          : `Status for project "${p.name}":`,
        { kind: "status", statusMap },
      ),
    );
  }

  async function handleFindingsCommand() {
    const p = getProject();
    const ready = p.connectors.filter((c) => c.status === "ready");

    if (ready.length === 0) {
      addMessage(
        assistantMsg(
          "No knowledge installed yet. Type 'install knowledge' to connect your data sources first.",
          {
            kind: "onboarding",
            onboardingConnectors:
              p.connectors.length > 0
                ? p.connectors
                : [
                    { connector: "github", uri: "", status: "idle" },
                    { connector: "jira", uri: "", status: "idle" },
                    { connector: "slack", uri: "", status: "idle" },
                  ],
          },
        ),
      );
      return;
    }

    const latest = ready[ready.length - 1];
    const loadId = makeId();
    // Token-based connectors (filesystem, github with GITHUB_TOKEN, jira) are fast.
    // Codex-routed connectors (slack, notion, sharepoint, google-drive) are slow (60-90s).
    const codexOnlyConnectors = new Set([
      "slack",
      "notion",
      "sharepoint",
      "googledrive",
    ]);
    const provider = codexOnlyConnectors.has(latest.connector)
      ? "codex"
      : "token";
    const estimatedWait =
      provider === "codex" ? " (may take ~60s via Codex)" : "";
    addMessage({
      id: loadId,
      role: "assistant",
      text: `Running analysis for ${latest.connector}${estimatedWait}…`,
      createdAt: now(),
      loading: true,
    });
    busy = true;

    try {
      const res = await postFindings({
        connector: latest.connector as ConnectorKind,
        uri: latest.uri,
        provider,
        role: "pmo",
        include_execution: false,
      });

      if (res.ok) {
        replaceMessage(
          loadId,
          assistantMsg(res.body.summary || "Analysis complete.", {
            kind: "findings",
            findingsResult: res.body,
          }),
        );
      } else {
        replaceMessage(
          loadId,
          assistantMsg(
            `Analysis failed: ${res.body?.error ?? "unknown error"}`,
          ),
        );
      }
    } catch (e) {
      replaceMessage(
        loadId,
        assistantMsg(`Error running analysis: ${String(e)}`),
      );
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head>
  <title>ContextOS</title>
</svelte:head>

<div class="app-shell">
  <!-- sidebar -->
  <aside class="sidebar" class:open={sidebarOpen}>
    <div class="sidebar-header">
      <span class="logo">ContextOS</span>
      <button
        class="sidebar-close"
        on:click={() => (sidebarOpen = false)}
        aria-label="close sidebar">✕</button
      >
    </div>

    <!-- project identity -->
    <div class="project-block">
      <p class="section-label">PROJECT</p>
      <button
        class="project-name"
        on:click={changeWorkspace}
        title="Click to change workspace path"
      >
        📁 {$project.name}
      </button>
      <p class="project-path">{$project.workspacePath}</p>
      {#if $project.knowledgeInstalledAt}
        <p class="installed-at">
          Knowledge installed {new Date(
            $project.knowledgeInstalledAt,
          ).toLocaleDateString()}
        </p>
      {/if}
    </div>

    <!-- connector knowledge status -->
    <div class="connectors-block">
      <p class="section-label">CONNECTORS</p>
      {#if $project.connectors.length === 0}
        <p class="empty-note">No connectors yet</p>
      {:else}
        {#each $project.connectors as ck}
          <div class="conn-item">
            <span
              class="dot"
              class:green={ck.status === "ready"}
              class:red={ck.status === "error"}
              class:yellow={ck.status === "ingesting"}
            />
            <span>{ck.connector}</span>
            <span class="conn-badge">{ck.status}</span>
          </div>
        {/each}
      {/if}
      <button class="install-btn" on:click={() => (showKnowledge = true)}>
        + Install Knowledge
      </button>
    </div>

    <!-- codex status -->
    <div class="codex-block">
      <p class="section-label">CODEX CLI</p>
      <div class="codex-row">
        <span
          class="dot"
          class:green={codexLoggedIn}
          class:red={!codexInstalled}
          class:yellow={codexInstalled && !codexLoggedIn}
        />
        <span
          >{codexInstalled
            ? codexLoggedIn
              ? `Logged in as ${codexAccount}`
              : "Not logged in"
            : "Not installed"}</span
        >
      </div>
    </div>

    <!-- nav -->
    <nav class="nav-links">
      <a href="/connectors">Connector debug ↗</a>
      <a href="/findings">Advanced findings ↗</a>
    </nav>
  </aside>

  <!-- main chat area -->
  <div class="chat-area">
    <header class="chat-header">
      <button
        class="menu-btn"
        on:click={() => (sidebarOpen = !sidebarOpen)}
        aria-label="menu">☰</button
      >
      <span class="header-title">{$project.name}</span>
      <div class="header-status">
        <span
          class="status-dot"
          class:green={apiStatus === "ok"}
          class:red={apiStatus === "unreachable"}
          title="API {apiStatus}"
        />
      </div>
    </header>

    {#if $chatMessages.length === 0}
      <div class="empty-state">
        <h2>Hello 👋</h2>
        <p>
          I'm your ContextOS assistant. Ask me about your project's delivery
          alignment, mismatches, or findings.
        </p>
        <div class="suggestions">
          <button
            on:click={() =>
              handleSend(new CustomEvent("send", { detail: "show findings" }))}
            >Show findings</button
          >
          <button
            on:click={() =>
              handleSend(new CustomEvent("send", { detail: "status" }))}
            >Check status</button
          >
          <button
            on:click={() => {
              showKnowledge = true;
            }}>Install knowledge</button
          >
          <button
            on:click={() =>
              handleSend(new CustomEvent("send", { detail: "help" }))}
            >Help</button
          >
        </div>
      </div>
    {:else}
      <ChatThread messages={$chatMessages} />
    {/if}

    <ChatInput on:send={handleSend} disabled={busy} />
  </div>
</div>

{#if showKnowledge}
  <KnowledgeInstall
    {codexLoggedIn}
    {codexPlugins}
    onClose={() => (showKnowledge = false)}
    on:done={() => {
      showKnowledge = false;
      addMessage(
        assistantMsg(
          "Knowledge installed! You can now ask me about your project. Try 'show findings'.",
        ),
      );
    }}
  />
{/if}

<style>
  :global(body) {
    margin: 0;
    font-family:
      system-ui,
      -apple-system,
      sans-serif;
    background: #f9fafb;
  }

  .app-shell {
    display: flex;
    height: 100vh;
    overflow: hidden;
  }

  /* ---- sidebar ---- */
  .sidebar {
    width: 240px;
    flex-shrink: 0;
    background: #111827;
    color: #e5e7eb;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    transition: transform 0.2s;
  }

  @media (max-width: 640px) {
    .sidebar {
      position: fixed;
      inset: 0 auto 0 0;
      z-index: 50;
      transform: translateX(-100%);
      width: 80vw;
    }
    .sidebar.open {
      transform: translateX(0);
    }
  }

  .sidebar-header {
    padding: 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid #374151;
  }
  .logo {
    font-weight: 700;
    font-size: 1rem;
    letter-spacing: 0.05em;
  }
  .sidebar-close {
    background: none;
    border: none;
    color: #9ca3af;
    cursor: pointer;
    display: none;
    font-size: 1rem;
  }
  @media (max-width: 640px) {
    .sidebar-close {
      display: block;
    }
  }

  .section-label {
    font-size: 0.65rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    color: #6b7280;
    padding: 0.75rem 1rem 0.25rem;
    margin: 0;
  }

  .project-block {
    padding: 0 1rem 0.75rem;
  }
  .project-name {
    background: none;
    border: none;
    color: #f9fafb;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    padding: 0;
    text-align: left;
    display: block;
    width: 100%;
  }
  .project-name:hover {
    color: #60a5fa;
  }
  .project-path {
    color: #6b7280;
    font-size: 0.72rem;
    margin: 2px 0 0;
    word-break: break-all;
  }
  .installed-at {
    color: #34d399;
    font-size: 0.7rem;
    margin: 3px 0 0;
  }

  .connectors-block {
    padding: 0 1rem 0.75rem;
  }
  .empty-note {
    color: #6b7280;
    font-size: 0.8rem;
    margin: 0.25rem 0;
  }
  .conn-item {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.82rem;
    margin-bottom: 0.25rem;
    color: #d1d5db;
  }
  .conn-badge {
    margin-left: auto;
    font-size: 0.65rem;
    background: #1f2937;
    color: #9ca3af;
    padding: 1px 5px;
    border-radius: 4px;
  }
  .install-btn {
    width: 100%;
    margin-top: 0.5rem;
    background: #1d4ed8;
    color: #fff;
    border: none;
    border-radius: 0.375rem;
    padding: 0.4rem 0.75rem;
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.15s;
  }
  .install-btn:hover {
    background: #1e40af;
  }

  .codex-block {
    padding: 0 1rem 0.75rem;
  }
  .codex-row {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.8rem;
    color: #d1d5db;
  }

  .dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .dot.green {
    background: #10b981;
  }
  .dot.red {
    background: #ef4444;
  }
  .dot.yellow {
    background: #f59e0b;
  }

  .nav-links {
    padding: 0.5rem 1rem 1rem;
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    border-top: 1px solid #374151;
  }
  .nav-links a {
    color: #9ca3af;
    font-size: 0.8rem;
    text-decoration: none;
  }
  .nav-links a:hover {
    color: #e5e7eb;
  }

  /* ---- chat area ---- */
  .chat-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    background: #fff;
  }

  .chat-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid #e5e7eb;
    background: #fff;
    position: sticky;
    top: 0;
    z-index: 10;
  }
  .menu-btn {
    background: none;
    border: none;
    font-size: 1.2rem;
    cursor: pointer;
    color: #6b7280;
    display: none;
  }
  @media (max-width: 640px) {
    .menu-btn {
      display: block;
    }
  }
  .header-title {
    font-weight: 600;
    font-size: 0.95rem;
    flex: 1;
  }
  .header-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .status-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  .status-dot.green {
    background: #10b981;
  }
  .status-dot.red {
    background: #ef4444;
  }

  /* empty state */
  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    padding: 2rem;
    color: #374151;
  }
  .empty-state h2 {
    font-size: 1.5rem;
    margin-bottom: 0.5rem;
  }
  .empty-state p {
    color: #6b7280;
    max-width: 380px;
    margin-bottom: 1.5rem;
  }
  .suggestions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    justify-content: center;
  }
  .suggestions button {
    background: #f3f4f6;
    border: 1px solid #e5e7eb;
    border-radius: 1rem;
    padding: 0.35rem 0.9rem;
    font-size: 0.85rem;
    cursor: pointer;
    transition: background 0.15s;
  }
  .suggestions button:hover {
    background: #e5e7eb;
  }
</style>
