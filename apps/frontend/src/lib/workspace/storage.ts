type BrowserStorage = Pick<Storage, "getItem" | "setItem" | "removeItem" | "key"> & {
  readonly length: number;
};

function isBrowserStorage(value: unknown): value is BrowserStorage {
  const storage = value as Partial<BrowserStorage> | null;
  return Boolean(
    storage &&
    typeof storage.getItem === "function" &&
    typeof storage.setItem === "function" &&
    typeof storage.removeItem === "function" &&
    typeof storage.key === "function" &&
    typeof storage.length === "number",
  );
}

export function getLocalStorage(): BrowserStorage | null {
  try {
    const storage =
      typeof globalThis === "undefined"
        ? undefined
        : (globalThis as { localStorage?: unknown }).localStorage;
    return isBrowserStorage(storage) ? storage : null;
  } catch {
    return null;
  }
}
