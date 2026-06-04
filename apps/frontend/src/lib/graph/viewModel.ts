import type { GraphEntity, GraphRelationship } from "$lib/types";

export type GraphLink = {
  id: string;
  source: string;
  target: string;
  label: string;
  strength: number;
  evidence?: string[];
};

export type GraphTypeSummary = {
  type: string;
  count: number;
  color: string;
};

export type EntityIndexItem = GraphEntity & {
  degree: number;
};

export type EntityIndexSection = {
  label: string;
  entities: EntityIndexItem[];
};

export type RelationshipRow = {
  entityName: string;
  confidence: number;
  evidence?: string[];
};

export type RelationshipKindGroup = {
  kind: string;
  incoming: RelationshipRow[];
  outgoing: RelationshipRow[];
};

export type FocusGraphRow = {
  id: string;
  side: "incoming" | "outgoing";
  y: number;
  entity: GraphEntity;
  link: GraphLink;
  color: string;
};

export function buildGraphLinks(
  entities: GraphEntity[],
  relationships: GraphRelationship[],
) {
  if (relationships.length > 0) {
    return relationships
      .filter(isSignalRelationship)
      .map((relationship) => ({
        id: relationship.id,
        source: relationship.from_id,
        target: relationship.to_id,
        label: relationship.kind,
        strength: relationship.confidence ?? 0.5,
        evidence: relationship.evidence,
      }))
      .sort(compareGraphLinks)
      .filter(
        (link) =>
          entities.some((entity) => entity.id === link.source) &&
          entities.some((entity) => entity.id === link.target),
      );
  }

  const links = new Map<string, GraphLink>();
  connectGroups(
    links,
    groupBy(entities, (entity) => entity.source || "unknown"),
    "source",
    0.8,
    4,
  );
  connectGroups(
    links,
    groupBy(entities, (entity) => normalizedEvidence(entity)),
    "evidence",
    0.55,
    3,
  );

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

export function linkDegree(links: GraphLink[]) {
  const degree = new Map<string, number>();
  for (const link of links) {
    degree.set(link.source, (degree.get(link.source) ?? 0) + 1);
    degree.set(link.target, (degree.get(link.target) ?? 0) + 1);
  }
  return degree;
}

export function buildGraphLegendTypes(
  entities: GraphEntity[],
): GraphTypeSummary[] {
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
  return [...counts.values()].sort(
    (a, b) => b.count - a.count || a.type.localeCompare(b.type),
  );
}

export function compareGraphEntities(
  a: GraphEntity & { degree: number },
  b: GraphEntity & { degree: number },
) {
  if (b.degree !== a.degree) return b.degree - a.degree;
  const confidenceDelta = (b.confidence ?? 0) - (a.confidence ?? 0);
  if (confidenceDelta !== 0) return confidenceDelta;
  return a.name.localeCompare(b.name);
}

export function topGraphEntity(
  entities: GraphEntity[],
  degree: Map<string, number>,
) {
  return (
    [...entities]
      .map((entity) => ({ ...entity, degree: degree.get(entity.id) ?? 0 }))
      .sort(compareGraphEntities)[0] ?? entities[0]
  );
}

export function linkedIdsForEntity(entityId: string, links: GraphLink[]) {
  const ids = new Set<string>();
  for (const link of links) {
    if (link.source === entityId) ids.add(link.target);
    if (link.target === entityId) ids.add(link.source);
  }
  return ids;
}

export function buildEntityIndexSections(
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

export function buildFocusGraphRows(
  entity: GraphEntity,
  links: GraphLink[],
  entitiesById: Map<string, GraphEntity>,
): FocusGraphRow[] {
  const incoming = buildSideRows(entity, links, entitiesById, "incoming");
  const outgoing = buildSideRows(entity, links, entitiesById, "outgoing");
  return [...positionFocusRows(incoming), ...positionFocusRows(outgoing)];
}

export function buildRelationshipGroups(
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

export function graphTypeColor(type: string) {
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

export function typeAccentStyle(type: string) {
  return `--type-color:${graphTypeColor(type || "entity")};`;
}

export function relationshipLabel(value: string) {
  return value.replaceAll("_", " ");
}

function isSignalRelationship(relationship: GraphRelationship) {
  return !(
    relationship.kind === "co_occurs_in_document" &&
    (relationship.confidence ?? 0) < 0.6
  );
}

function compareGraphLinks(a: GraphLink, b: GraphLink) {
  const priority = relationshipPriority(b.label) - relationshipPriority(a.label);
  if (priority !== 0) return priority;
  return b.strength - a.strength;
}

function relationshipPriority(label: string) {
  return label === "co_occurs_in_document" ? 0 : 1;
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
    .sort(
      (a, b) =>
        b.link.strength - a.link.strength ||
        a.entity.name.localeCompare(b.entity.name),
    )
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

function groupBy(
  entities: GraphEntity[],
  keyFn: (entity: GraphEntity) => string,
) {
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
