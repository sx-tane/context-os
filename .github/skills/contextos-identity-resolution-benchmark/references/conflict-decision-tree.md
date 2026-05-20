# Conflict Decision Tree

Use this tree whenever two layers produce different merge decisions for the same pair.

## Step 1: Is one result from Layer 5 (human confirmation)?
- Yes → accept the human decision, log the divergence, and update the layer that was wrong.
- No → continue to Step 2.

## Step 2: Is one result from Layer 4 (historical memory)?
- Yes, and the current evidence (Layer 1–3) agrees → accept Layer 4, no action needed.
- Yes, but current evidence contradicts → prefer current evidence, log divergence as "rename or drift", flag for human review.
- No → continue to Step 3.

## Step 3: Do Layer 2 (semantic) and Layer 3 (relationship) conflict?
- Semantic says merge, relationship says separate → keep separate, flag as "semantic false positive", add relationship-based block rule candidate.
- Relationship says merge, semantic says separate → accept relationship merge with lower confidence, add to Layer 5 queue if impact is high.
- Both say merge with conflicting canonical targets → escalate to Layer 5.

## Step 4: Low confidence in all layers?
- Confidence below threshold in all layers → mark as unresolved, emit needs-review event, do not merge.

## Routing Summary

| Situation | Action |
|-----------|--------|
| Human decision available | Accept, log divergence |
| History vs current conflict | Prefer current, flag drift |
| Semantic contradicts relationship | Keep separate, add block candidate |
| All layers low confidence | Mark unresolved, needs-review event |
| Two layers conflict on canonical target | Escalate to human confirmation |
