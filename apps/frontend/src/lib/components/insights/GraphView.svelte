<script lang="ts">
    import type { GraphData, GraphEntity } from "$lib/types";
    import {
        buildEntityIndexSections,
        buildFocusGraphRows,
        buildGraphLegendTypes,
        buildGraphLinks,
        buildRelationshipGroups,
        linkDegree,
        linkedIdsForEntity,
        relationshipLabel,
        topGraphEntity,
        typeAccentStyle,
    } from "$lib/graphViewModel";

    export let graphData: GraphData | null = null;
    export let selectedEntity: GraphEntity | null = null;
    export let hasSources = false;

    let entityQuery = "";

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
</script>

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

<style>
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

    .entity-index::-webkit-scrollbar {
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
    .entity-row.selected,
    .entity-row.linked:not(.selected) {
        border-left-color: transparent;
        background: transparent;
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
        left: 50%;
        top: 50%;
        transform: translate(-50%, -50%);
        border: 1px solid rgba(215, 210, 200, 0.65);
        border-radius: 8px;
        background: rgba(248, 246, 239, 0.92);
        padding: 18px;
        text-align: center;
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

    .node-legend i {
        width: 9px;
        height: 9px;
        border-radius: 50%;
        background: var(--type-color);
        display: inline-block;
        flex: 0 0 auto;
    }

    .node-legend b {
        color: #8a8678;
        font-weight: 400;
    }

    @media (max-width: 1100px) {
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
    }

    @media (max-width: 760px) {
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
