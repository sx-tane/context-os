<script lang="ts">
    import { getGraphData } from "$lib/api";
    import type { GraphData, GraphEntity } from "$lib/types";

    export let workspacePath: string = "";
    export let open: boolean = false;

    let data: GraphData | null = null;
    let loading = false;
    let error = "";
    let filterType = "";

    async function load() {
        if (!workspacePath) return;
        loading = true;
        error = "";
        data = await getGraphData(workspacePath, filterType || undefined);
        if (!data) error = "Could not load graph data. Run a connector first.";
        loading = false;
    }

    $: if (open && workspacePath) {
        load();
    }

    function entityTypeColor(type: string): string {
        switch (type) {
            case "feature":
                return "bg-blue-100 text-blue-800";
            case "service":
                return "bg-green-100 text-green-800";
            case "requirement":
                return "bg-yellow-100 text-yellow-800";
            case "person":
                return "bg-purple-100 text-purple-800";
            case "api":
                return "bg-orange-100 text-orange-800";
            default:
                return "bg-gray-100 text-gray-700";
        }
    }

    function confidenceColor(conf: number): string {
        const pct = Math.round((conf ?? 0) * 100);
        if (pct >= 85) return "bg-green-400";
        if (pct >= 60) return "bg-yellow-400";
        return "bg-red-400";
    }
</script>

{#if open}
    <div class="flex flex-col h-full overflow-hidden">
        <!-- Header -->
        <div
            class="flex items-center justify-between px-4 py-3 border-b bg-white"
        >
            <h2 class="text-sm font-semibold text-gray-800">Entity Graph</h2>
            <button
                class="text-xs text-gray-500 hover:text-gray-800 underline"
                on:click={load}
            >
                Refresh
            </button>
        </div>

        <!-- Filter bar -->
        <div class="flex gap-2 px-4 py-2 border-b bg-gray-50">
            <select
                bind:value={filterType}
                on:change={load}
                class="text-xs border rounded px-2 py-1 bg-white text-gray-700 flex-1"
            >
                <option value="">All types</option>
                <option value="feature">Feature</option>
                <option value="service">Service</option>
                <option value="requirement">Requirement</option>
                <option value="person">Person</option>
                <option value="api">API</option>
            </select>
            {#if data}
                <span class="text-xs text-gray-500 self-center">
                    {data.count} entit{data.count === 1 ? "y" : "ies"}
                </span>
            {/if}
        </div>

        <!-- Entity list -->
        <div class="flex-1 overflow-y-auto px-4 py-3 space-y-2">
            {#if loading}
                <p class="text-xs text-gray-400 text-center py-8">Loading…</p>
            {:else if error}
                <p class="text-xs text-red-500 text-center py-8">{error}</p>
            {:else if !data || data.entities.length === 0}
                <p class="text-xs text-gray-400 text-center py-8">
                    No entities found. Run a connector to populate the graph.
                </p>
            {:else}
                {#each data.entities as entity (entity.id)}
                    {@const pct = Math.round((entity.confidence ?? 0) * 100)}
                    <div
                        class="rounded border bg-white shadow-sm p-3 space-y-1"
                    >
                        <div class="flex items-start justify-between gap-2">
                            <span
                                class="text-sm font-medium text-gray-800 truncate flex-1"
                                >{entity.name}</span
                            >
                            <span
                                class="text-xs px-1.5 py-0.5 rounded shrink-0 {entityTypeColor(
                                    entity.type,
                                )}">{entity.type}</span
                            >
                        </div>
                        <div class="flex items-center gap-2">
                            <div class="h-1.5 rounded-full bg-gray-200 flex-1">
                                <div
                                    class="h-1.5 rounded-full {confidenceColor(
                                        entity.confidence,
                                    )}"
                                    style="width: {pct}%"
                                ></div>
                            </div>
                            <span class="text-xs text-gray-500 tabular-nums"
                                >{pct}%</span
                            >
                        </div>
                        {#if entity.needs_human}
                            <p class="text-xs text-amber-600">
                                ⚠ Review needed{entity.conflict_reason
                                    ? `: ${entity.conflict_reason}`
                                    : ""}
                            </p>
                        {/if}
                        {#if entity.aliases && entity.aliases.length > 0}
                            <p class="text-xs text-gray-400">
                                {entity.aliases.length} alias{entity.aliases
                                    .length === 1
                                    ? ""
                                    : "es"}
                            </p>
                        {/if}
                    </div>
                {/each}
            {/if}
        </div>
    </div>
{/if}
