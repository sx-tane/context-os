import { composerHeight } from "../chat/composer";

describe("composerHeight", () => {
  it("uses the content height when it is inside min and max bounds", () => {
    expect(composerHeight(180, 400, 42)).toBe(180);
  });

  it("clamps short content to the minimum composer height", () => {
    expect(composerHeight(10, 400, 34)).toBe(34);
  });

  it("clamps long content to the available composer height", () => {
    expect(composerHeight(900, 320, 42)).toBe(320);
  });

  it("does not return less than the minimum when max is smaller than min", () => {
    expect(composerHeight(900, 20, 34)).toBe(34);
  });
});
