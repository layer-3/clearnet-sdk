// Gateway client errors. Mirrors APIError / SubmissionError in pkg/client.

export type GatewayErrorKind =
  | "network" // gateway unreachable, timed out, or response unreadable
  | "identity" // gateway serves a different network than expected
  | "trust" // gateway identity differs from the pinned fingerprint
  | "capability" // gateway lacks a required capability
  | "input"; // invalid caller input

export class GatewayError extends Error {
  readonly kind: GatewayErrorKind;

  constructor(kind: GatewayErrorKind, message: string, cause?: unknown) {
    super(message, cause === undefined ? undefined : { cause });
    this.name = "GatewayError";
    this.kind = kind;
  }
}

export interface APIErrorInit {
  httpStatus: number;
  code?: number;
  message: string;
  field?: string;
  data?: unknown;
  /** False when the body did not carry the REST error envelope. */
  recognized?: boolean;
}

/** A non-2xx gateway response carrying the REST error envelope. */
export class APIError extends Error {
  readonly httpStatus: number;
  readonly code: number;
  readonly field: string | undefined;
  readonly data: unknown;
  readonly recognized: boolean;

  constructor(init: APIErrorInit) {
    super(init.message);
    this.name = "APIError";
    this.httpStatus = init.httpStatus;
    this.code = init.code ?? 0;
    this.field = init.field;
    this.data = init.data;
    this.recognized = init.recognized ?? true;
  }
}

/**
 * Always carries the precomputed operation ID. `outcomeUnknown` is never a
 * licence to sign the transfer again at a new nonce.
 */
export class SubmissionError extends Error {
  /** Whether the POST was attempted. False means this call failed in local preflight. */
  readonly submitted: boolean;
  readonly operationId: string;
  readonly outcomeUnknown: boolean;

  constructor(init: { submitted: boolean; operationId: string; outcomeUnknown: boolean; cause: unknown }) {
    super(init.cause instanceof Error ? init.cause.message : String(init.cause), { cause: init.cause });
    this.name = "SubmissionError";
    this.submitted = init.submitted;
    this.operationId = init.operationId;
    this.outcomeUnknown = init.outcomeUnknown;
  }

  /** The gateway's definite rejection, if this was one (CLI: !OutcomeUnknown with a recognized code). */
  get definiteRejection(): APIError | undefined {
    const api = this.cause instanceof APIError ? this.cause : undefined;
    if (this.submitted && !this.outcomeUnknown && api && api.code !== 0 && api.recognized) return api;
    return undefined;
  }
}

/** JSON-RPC-compatible codes shared by REST and /rpc (gateway errors.go). */
export const GATEWAY_ERROR_CODES = {
  invalidRequest: -32600,
  invalidParams: -32602,
  insufficientBalance: -32000,
  invalidSignature: -32001,
  nonceMismatch: -32002,
  operationRejected: -32003,
  outcomeUnknown: -32006,
  indexUnavailable: -32007,
  backendUnavailable: -32008,
  operationConflict: -32009,
} as const;
