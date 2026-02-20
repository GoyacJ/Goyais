import { validateEventEnvelope } from "@goyais/protocol/src/validators";

export function assertEventEnvelope(payload: unknown): boolean {
  return validateEventEnvelope(payload);
}
