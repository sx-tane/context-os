<script lang="ts">
    import type {
        ApiErrorBody,
        ConnectorKind,
        FindingsResult,
        IngestProvider,
        PresentationRole,
    } from "$lib/types";
    import { postFindings } from "$lib/api";

    const connectors: ConnectorKind[] = [
        "filesystem",
        "github",
        "jira",
        "slack",
        "googledrive",
        "notion",
        "sharepoint",
    ];
    const roles: PresentationRole[] = [
        "pmo",
        "presentation_layer",
        "service_layer",
        "qa",
        "architecture",
    ];

    let connector: ConnectorKind = "filesystem";
    let provider: IngestProvider = "token";
    let role: PresentationRole = "pmo";
    let uri = "inline.txt";
    let token = "";
    let content =
        "frontend expects refundStatus but backend exposes missingRefundState";
    let includeExecution = true;

    let loading = false;
    let error = "";
    let result: FindingsResult | null = null;

    async function runFindings() {
        loading = true;
        error = "";
        result = null;
        try {
            const response = await postFindings({
                connector,
                provider,
                role,
                uri,
                content,
                token: token.trim() || undefined,
                include_execution: includeExecution,
            });
            if (!response.ok) {
                const body = response.body as ApiErrorBody;
                error = body.message ?? body.error ?? "Request failed";
                return;
            }
            result = response.body;
        } catch (e) {
            error = String(e);
        } finally {
            loading = false;
        }
    }

    $: selectedView = result ? result.views?.[role] : null;
</script>

<svelte:head>
    <title>ContextOS Findings</title>
</svelte:head>

<main>
    <header>
        <h1>ContextOS Findings</h1>
        <p>
            Graph-backed role summaries for delivery intelligence. <a href="/"
                >Open Debug Connectors</a
            >
        </p>
    </header>

    <section class="card">
        <h2>Run Analysis</h2>
        <label>
            Connector
            <select bind:value={connector}>
                {#each connectors as item}
                    <option value={item}>{item}</option>
                {/each}
            </select>
        </label>

        <label>
            Provider
            <select bind:value={provider}>
                <option value="token">token</option>
                <option value="codex">codex</option>
            </select>
        </label>

        <label>
            Role
            <select bind:value={role}>
                {#each roles as item}
                    <option value={item}>{item}</option>
                {/each}
            </select>
        </label>

        <label>
            URI
            <input bind:value={uri} placeholder="inline.txt or connector URI" />
        </label>

        <label>
            Token (optional)
            <input bind:value={token} placeholder="provider token" />
        </label>

        <label>
            Content
            <textarea
                bind:value={content}
                rows="5"
                placeholder="Optional inline content for local analysis"
            />
        </label>

        <label class="toggle">
            <input type="checkbox" bind:checked={includeExecution} />
            Include hidden assistive execution metadata
        </label>

        <button on:click={runFindings} disabled={loading}>
            {loading ? "Running..." : "Run Findings"}
        </button>
    </section>

    {#if error}
        <section class="card error">{error}</section>
    {/if}

    {#if result}
        <section class="card">
            <h2>Summary</h2>
            <p><strong>Trace:</strong> {result.trace_id}</p>
            <p><strong>Role:</strong> {result.role}</p>
            <p><strong>Mismatches:</strong> {result.mismatch_count}</p>
            <p>
                <strong>Severity:</strong> high {result.severity_count?.high ?? 0},
                medium {result.severity_count?.medium ?? 0}, low {result
                    .severity_count?.low ?? 0}
            </p>
            <pre>{result.summary}</pre>
        </section>

        {#if selectedView}
            <section class="card">
                <h2>Role View: {selectedView.role}</h2>
                <p>{selectedView.summary}</p>
                {#if selectedView.next_actions.length > 0}
                    <h3>Next Actions</h3>
                    <ul>
                        {#each selectedView.next_actions as action}
                            <li>{action}</li>
                        {/each}
                    </ul>
                {/if}
            </section>
        {/if}

        <section class="card">
            <h2>PMO View Model</h2>
            <p><strong>Facts:</strong> {result.pmo?.facts.length ?? 0}</p>
            <p><strong>Risks:</strong> {result.pmo?.risks.length ?? 0}</p>
            <p>
                <strong>Impacts:</strong>
                {result.pmo?.impacts.join(", ") || "none"}
            </p>
            <h3>Recommended Decisions</h3>
            <ul>
                {#each result.pmo?.recommended_decisions ?? [] as decision}
                    <li>{decision}</li>
                {/each}
            </ul>
        </section>

        <section class="card">
            <h2>Assistive Execution</h2>
            <p>
                <strong>Enabled:</strong>
                {result.execution?.enabled ? "yes" : "no"}
            </p>
            <p>
                <strong>Assistive:</strong>
                {result.execution?.assistive ? "yes" : "no"}
            </p>
            <p><strong>Summary:</strong> {result.execution?.summary ?? "none"}</p>
            {#if result.execution?.error}
                <p class="error">{result.execution.error}</p>
            {/if}
        </section>

        <section class="card">
            <h2>Mismatches</h2>
            {#if (result.mismatches?.length ?? 0) === 0}
                <p>No mismatches detected.</p>
            {:else}
                <ul>
                    {#each result.mismatches ?? [] as mismatch}
                        <li>
                            <strong
                                >[{mismatch.severity ?? "review"}] {mismatch.summary ?? mismatch.description ?? mismatch.type ?? mismatch.id}</strong
                            >
                            <div>id: {mismatch.id}</div>
                            <div>confidence: {mismatch.confidence}</div>
                            <div>impact: {mismatch.impact}</div>
                            <div>
                                evidence: {mismatch.evidence?.join(", ") ||
                                    "none"}
                            </div>
                            <div>
                                recommended: {mismatch.recommended ?? mismatch.recommended_action ?? "none"}
                            </div>
                        </li>
                    {/each}
                </ul>
            {/if}
        </section>
    {/if}
</main>

<style>
    main {
        font-family: system-ui, sans-serif;
        max-width: 860px;
        margin: 2rem auto;
        padding: 0 1rem 2rem;
    }

    header h1 {
        margin-bottom: 0.4rem;
    }

    .card {
        border: 1px solid #e5e7eb;
        border-radius: 12px;
        padding: 1rem;
        margin-bottom: 1rem;
        background: #fff;
    }

    label {
        display: block;
        margin: 0.5rem 0;
        font-size: 0.92rem;
    }

    input,
    textarea,
    select {
        width: 100%;
        margin-top: 0.25rem;
        padding: 0.5rem;
        box-sizing: border-box;
        border: 1px solid #d1d5db;
        border-radius: 8px;
    }

    .toggle {
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    .toggle input {
        width: auto;
        margin: 0;
    }

    button {
        margin-top: 0.6rem;
        border: 0;
        border-radius: 8px;
        padding: 0.55rem 0.9rem;
        background: #111827;
        color: #fff;
        cursor: pointer;
    }

    button:disabled {
        opacity: 0.6;
        cursor: default;
    }

    pre {
        background: #f8fafc;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        padding: 0.75rem;
        white-space: pre-wrap;
    }

    .error {
        color: #b91c1c;
    }

    @media (max-width: 768px) {
        main {
            margin: 1rem auto;
            padding: 0 0.75rem 1.5rem;
        }

        .card {
            padding: 0.85rem;
        }
    }
</style>
