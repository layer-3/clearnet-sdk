import { describe, expect, it } from "vitest";

import {
  normalizeMinConfirmations,
  normalizeReceiptTimeoutMs,
} from "../../src/core/validation.js";

describe("core validation helpers", () => {
  it("normalizes min confirmations and rejects unsafe numeric depths", () => {
    expect(normalizeMinConfirmations(0)).toBe(0n);
    expect(normalizeMinConfirmations(1n << 80n)).toBe(1n << 80n);

    for (const value of [-1, 1.5, Number.MAX_SAFE_INTEGER + 1]) {
      expect(() => normalizeMinConfirmations(value)).toThrowError(
        expect.objectContaining({
          code: "INVALID_CONFIRMATIONS",
          message: "minConfirmations must be a non-negative safe integer",
        }),
      );
    }

    expect(() => normalizeMinConfirmations(-1n)).toThrowError(
      expect.objectContaining({
        code: "INVALID_CONFIRMATIONS",
        message: "minConfirmations must be non-negative",
      }),
    );
  });

  it("normalizes receipt timeouts and rejects invalid millisecond values", () => {
    expect(normalizeReceiptTimeoutMs(1)).toBe(1);
    expect(normalizeReceiptTimeoutMs(30_000)).toBe(30_000);

    for (const value of [0, -1, 1.5, Number.MAX_SAFE_INTEGER + 1]) {
      expect(() => normalizeReceiptTimeoutMs(value)).toThrowError(
        expect.objectContaining({
          code: "RECEIPT_TIMEOUT",
          message: "receiptTimeoutMs must be a positive safe integer",
        }),
      );
    }
  });
});
