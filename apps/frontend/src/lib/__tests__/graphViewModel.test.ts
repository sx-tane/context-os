import {
  buildEntityIndexSections,
  buildGraphLegendTypes,
  buildGraphLinks,
  linkDegree,
  linkedIdsForEntity,
  relationshipLabel,
  topGraphEntity,
} from "../graph/viewModel";

import type { GraphEntity, GraphRelationship } from "../types";

describe("buildGraphLinks — relationship data", () => {
  it("keeps only relationships that reference known entities and sorts by confidence", () => {
    const entities = makeEntities();
    const relationships: GraphRelationship[] = [
      {
        id: "low",
        from_id: "feature",
        to_id: "api",
        kind: "depends_on",
        confidence: 0.4,
      },
      {
        id: "high",
        from_id: "feature",
        to_id: "owner",
        kind: "owned_by",
        confidence: 0.9,
      },
      {
        id: "missing",
        from_id: "feature",
        to_id: "missing",
        kind: "invalid",
        confidence: 1,
      },
    ];

    const links = buildGraphLinks(entities, relationships);

    expect(links.map((link) => link.id)).toEqual(["high", "low"]);
    expect(links[0]).toMatchObject({
      source: "feature",
      target: "owner",
      label: "owned_by",
      strength: 0.9,
    });
  });

  it("filters low-confidence co-occurrence links and prioritizes typed delivery relationships", () => {
    const entities = makeEntities();
    const relationships: GraphRelationship[] = [
      {
        id: "co",
        from_id: "feature",
        to_id: "api",
        kind: "co_occurs_in_document",
        confidence: 0.95,
      },
      {
        id: "noisy",
        from_id: "feature",
        to_id: "owner",
        kind: "co_occurs_in_document",
        confidence: 0.5,
      },
      {
        id: "typed",
        from_id: "feature",
        to_id: "refund",
        kind: "requirement_affects_api",
        confidence: 0.7,
      },
    ];

    const links = buildGraphLinks(entities, relationships);

    expect(links.map((link) => link.id)).toEqual(["typed", "co"]);
  });
});

describe("buildGraphLinks — inferred data", () => {
  it("infers links from shared source, evidence, and aliases when explicit relationships are absent", () => {
    const links = buildGraphLinks(
      [
        ...makeEntities(),
        {
          id: "doc-a",
          name: "Document A",
          type: "note",
          source: "docs-a",
          confidence: 0.5,
          evidence: ["shared-only evidence"],
        },
        {
          id: "doc-b",
          name: "Document B",
          type: "note",
          source: "docs-b",
          confidence: 0.4,
          evidence: ["shared-only evidence"],
        },
      ],
      [],
    );

    expect(links.some((link) => link.label === "source")).toBe(true);
    expect(links.some((link) => link.label === "evidence")).toBe(true);
    expect(links.some((link) => link.label === "alias")).toBe(true);
  });
});

describe("buildEntityIndexSections — selected context", () => {
  it("prioritizes selected, linked, and top entities", () => {
    const entities = makeEntities();
    const links = buildGraphLinks(entities, [
      {
        id: "rel",
        from_id: "feature",
        to_id: "api",
        kind: "depends_on",
        confidence: 0.8,
      },
    ]);
    const degree = linkDegree(links);
    const linked = linkedIdsForEntity("feature", links);

    const sections = buildEntityIndexSections(
      entities,
      degree,
      entities[0],
      linked,
      "",
    );

    expect(sections.map((section) => section.label)).toEqual([
      "Selected",
      "Linked",
      "Top entities",
    ]);
    expect(sections[0].entities[0].id).toBe("feature");
    expect(sections[1].entities[0].id).toBe("api");
  });

  it("returns filtered matches when a search query is present", () => {
    const sections = buildEntityIndexSections(
      makeEntities(),
      new Map(),
      null,
      new Set(),
      "owner",
    );

    expect(sections).toHaveLength(1);
    expect(sections[0].label).toBe("Matches");
    expect(sections[0].entities.map((entity) => entity.id)).toEqual(["owner"]);
  });
});

describe("topGraphEntity", () => {
  it("chooses the highest degree entity before confidence and name", () => {
    const entities = makeEntities();
    const degree = new Map([
      ["feature", 1],
      ["api", 4],
      ["owner", 2],
    ]);

    expect(topGraphEntity(entities, degree).id).toBe("api");
  });
});

describe("buildGraphLegendTypes", () => {
  it("summarizes entity types by count", () => {
    const legend = buildGraphLegendTypes(makeEntities());

    expect(legend.map((item) => [item.type, item.count])).toEqual([
      ["feature", 2],
      ["person", 1],
      ["service", 1],
    ]);
    expect(legend.every((item) => item.color.startsWith("#"))).toBe(true);
  });
});

describe("relationshipLabel", () => {
  it("renders relationship kinds without underscores", () => {
    expect(relationshipLabel("depends_on")).toBe("depends on");
  });
});

function makeEntities(): GraphEntity[] {
  return [
    {
      id: "feature",
      name: "Checkout",
      type: "feature",
      source: "jira",
      confidence: 0.9,
      evidence: ["shared evidence"],
      aliases: ["Refund"],
    },
    {
      id: "api",
      name: "Payments API",
      type: "service",
      source: "github",
      confidence: 0.8,
      evidence: ["shared evidence"],
      aliases: ["Refund"],
    },
    {
      id: "owner",
      name: "Service Owner",
      type: "person",
      source: "slack",
      confidence: 0.7,
      evidence: ["ownership thread"],
    },
    {
      id: "refund",
      name: "Refund Status",
      type: "feature",
      source: "jira",
      confidence: 0.6,
      evidence: ["refund notes"],
    },
  ];
}
