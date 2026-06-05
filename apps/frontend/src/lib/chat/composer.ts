export function composerHeight(
  scrollHeight: number,
  maxHeight: number,
  minHeight: number,
) {
  const safeScrollHeight = Math.max(0, scrollHeight);
  const safeMinHeight = Math.max(0, minHeight);
  const safeMaxHeight = Math.max(safeMinHeight, maxHeight);
  return Math.min(Math.max(safeScrollHeight, safeMinHeight), safeMaxHeight);
}
