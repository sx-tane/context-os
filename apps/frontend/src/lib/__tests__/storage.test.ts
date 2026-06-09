import { getLocalStorage } from "$lib/workspace/storage";

function clearStorageGlobal(): void {
  delete (globalThis as { localStorage?: unknown }).localStorage;
}

beforeEach(() => {
  clearStorageGlobal();
});

afterEach(() => {
  clearStorageGlobal();
});

describe("getLocalStorage", () => {
  it("returns null when localStorage is not a Storage object", () => {
    (globalThis as { localStorage?: unknown }).localStorage = {
      getItem: "not-a-function",
    };

    expect(getLocalStorage()).toBeNull();
  });

  it("returns browser storage when the storage shape is complete", () => {
    const storage = {
      getItem: jest.fn(),
      setItem: jest.fn(),
      removeItem: jest.fn(),
      key: jest.fn(),
      length: 0,
    };
    (globalThis as { localStorage?: unknown }).localStorage = storage;

    expect(getLocalStorage()).toBe(storage);
  });
});
