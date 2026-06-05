<script lang="ts">
    import type { ChatMessage } from "$lib/types";
    import {
        reviewCandidateCount,
        topActionableFindings,
    } from "$lib/findings/viewModel";

    export let message: ChatMessage;

    const isUser = message.role === "user";
    const isSystem = message.role === "system";
    const hasCard = !!message.card;
    const card = message.card;

    function severityClass(s?: string): string {
        if (s === "critical" || s === "high") return "sev-high";
        if (s === "medium") return "sev-med";
        return "sev-low";
    }
</script>

<div
    class="msg-row"
    class:user={isUser}
    class:assistant={!isUser && !isSystem}
    class:system={isSystem}
>
    {#if !isUser && !isSystem}
        <span class="avatar">🤖</span>
    {/if}

    <div class="bubble">
        {#if message.loading}
            <span class="loading-dots"><span></span><span></span><span></span></span>
        {:else}
            <p class="text">{message.text}</p>

            {#if hasCard && card}
                {#if card.kind === "status" && card.statusMap}
                    <div class="card status-card">
                        <h4>Connector Status</h4>
                        <ul>
                            {#each Object.entries(card.statusMap) as [name, ok]}
                                <li>
                                    <span
                                        class="dot"
                                        class:green={ok}
                                        class:red={!ok}
                                    ></span>
                                    {name}
                                    <span class="badge"
                                        >{ok ? "ready" : "not configured"}</span
                                    >
                                </li>
                            {/each}
                        </ul>
                    </div>
                {/if}

                {#if card.kind === "ingest" && card.ingestResult}
                    {@const r = card.ingestResult}
                    <div class="card ingest-card">
                        <h4>Ingestion complete</h4>
                        <p>
                            {r.event_count ?? 0} events · trace
                            <code>{r.event?.trace_id ?? "—"}</code>
                        </p>
                    </div>
                {/if}

                {#if card.kind === "findings" && card.findingsResult}
                    {@const f = card.findingsResult}
                    {@const topFindings = topActionableFindings(f, 3)}
                    <div class="card findings-card">
                        <h4>Findings — {f.role}</h4>
                        <p class="summary">{f.summary}</p>
                        <p>
                            <strong>{f.mismatch_count ?? topFindings.length}</strong> top issues ·
                            {reviewCandidateCount(f)} review candidates
                        </p>
                        {#if topFindings.length}
                            <ul class="mismatches">
                                {#each topFindings as m}
                                    <li class={severityClass(m.severity)}>
                                        <strong>{m.severity ?? "review"}</strong> · {m.summary ?? m.description ?? m.type ?? m.id}
                                        {#if m.recommended}
                                            <span class="rec"
                                                >→ {m.recommended}</span
                                            >
                                        {/if}
                                    </li>
                                {/each}
                            </ul>
                        {/if}
                    </div>
                {/if}

                {#if card.kind === "onboarding" && card.onboardingConnectors}
                    <div class="card onboarding-card">
                        <h4>Connector readiness</h4>
                        <ul>
                            {#each card.onboardingConnectors as c}
                                <li>
                                    <span
                                        class="dot"
                                        class:green={c.status === "ready"}
                                        class:yellow={c.status ===
                                            "ingesting" ||
                                            c.status === "configuring"}
                                        class:red={c.status === "error" ||
                                            c.status === "idle"}
                                    ></span>
                                    <strong>{c.connector}</strong>
                                    {#if c.uri}<span class="uri">{c.uri}</span
                                        >{/if}
                                    <span class="badge">{c.status}</span>
                                </li>
                            {/each}
                        </ul>
                    </div>
                {/if}
            {/if}
        {/if}
    </div>

    {#if isUser}
        <span class="avatar user-av">🧑</span>
    {/if}
</div>

<style>
    .msg-row {
        display: flex;
        align-items: flex-start;
        gap: 0.5rem;
        margin-bottom: 0.75rem;
    }
    .msg-row.user {
        flex-direction: row-reverse;
    }
    .msg-row.system {
        justify-content: center;
    }

    .avatar {
        font-size: 1.25rem;
        flex-shrink: 0;
        margin-top: 0.15rem;
    }

    .bubble {
        max-width: 72%;
        background: #f3f4f6;
        border-radius: 1rem;
        padding: 0.6rem 0.9rem;
        font-size: 0.9rem;
        line-height: 1.5;
    }
    .user .bubble {
        background: #2563eb;
        color: #fff;
    }
    .system .bubble {
        background: transparent;
        color: #6b7280;
        font-size: 0.8rem;
        padding: 0.2rem 0.5rem;
        border-radius: 0;
    }

    .text {
        margin: 0;
        white-space: pre-wrap;
    }

    /* loading dots */
    .loading-dots {
        display: flex;
        gap: 4px;
        padding: 0.2rem 0;
    }
    .loading-dots span {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: #9ca3af;
        animation: bounce 1.2s infinite ease-in-out;
    }
    .loading-dots span:nth-child(2) {
        animation-delay: 0.2s;
    }
    .loading-dots span:nth-child(3) {
        animation-delay: 0.4s;
    }
    @keyframes bounce {
        0%,
        80%,
        100% {
            transform: scale(0.8);
        }
        40% {
            transform: scale(1.2);
        }
    }

    /* cards */
    .card {
        margin-top: 0.6rem;
        background: #fff;
        border: 1px solid #e5e7eb;
        border-radius: 0.5rem;
        padding: 0.6rem 0.8rem;
        font-size: 0.82rem;
    }
    .card h4 {
        margin: 0 0 0.4rem;
        font-size: 0.85rem;
        color: #111827;
    }
    .card ul {
        margin: 0;
        padding-left: 1rem;
    }
    .card li {
        margin-bottom: 0.25rem;
    }
    .card p {
        margin: 0.2rem 0;
        color: #374151;
    }
    .card code {
        font-size: 0.78rem;
        background: #f3f4f6;
        padding: 0 3px;
        border-radius: 3px;
    }

    .dot {
        display: inline-block;
        width: 8px;
        height: 8px;
        border-radius: 50%;
        margin-right: 4px;
        vertical-align: middle;
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

    .badge {
        margin-left: 4px;
        font-size: 0.72rem;
        background: #f3f4f6;
        color: #6b7280;
        padding: 1px 5px;
        border-radius: 4px;
    }
    .uri {
        color: #6b7280;
        font-size: 0.78rem;
        margin-left: 4px;
    }

    .mismatches {
        padding-left: 0;
        list-style: none;
    }
    .mismatches li {
        padding: 0.25rem 0;
        border-bottom: 1px solid #f3f4f6;
        font-size: 0.8rem;
    }
    .sev-high strong {
        color: #dc2626;
    }
    .sev-med strong {
        color: #d97706;
    }
    .sev-low strong {
        color: #16a34a;
    }
    .rec {
        display: block;
        color: #2563eb;
        font-size: 0.78rem;
        margin-top: 2px;
    }

    .summary {
        color: #374151;
        font-style: italic;
    }
</style>
