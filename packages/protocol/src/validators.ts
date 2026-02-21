import Ajv from "ajv";
import addFormats from "ajv-formats";
import envelopeSchema from "../schemas/v2/event-envelope.schema.json";
import executionCreateRequestSchema from "../schemas/v2/execution-create.request.schema.json";
import executionCreateResponseSchema from "../schemas/v2/execution-create.response.schema.json";
import confirmationDecisionRequestSchema from "../schemas/v2/confirmation-decision.request.schema.json";
import confirmationDecisionResponseSchema from "../schemas/v2/confirmation-decision.response.schema.json";
import goyaisErrorSchema from "../schemas/v2/goyais-error.schema.json";
import eventPayloadPlanSchema from "../schemas/v2/event-payload-plan.schema.json";
import eventPayloadToolCallSchema from "../schemas/v2/event-payload-tool-call.schema.json";
import eventPayloadToolResultSchema from "../schemas/v2/event-payload-tool-result.schema.json";
import eventPayloadPatchSchema from "../schemas/v2/event-payload-patch.schema.json";
import eventPayloadErrorSchema from "../schemas/v2/event-payload-error.schema.json";
import eventPayloadDoneSchema from "../schemas/v2/event-payload-done.schema.json";

const ajv = new Ajv({ allErrors: true, strict: false });
addFormats(ajv);

[
  goyaisErrorSchema,
  eventPayloadPlanSchema,
  eventPayloadToolCallSchema,
  eventPayloadToolResultSchema,
  eventPayloadPatchSchema,
  eventPayloadErrorSchema,
  eventPayloadDoneSchema,
  executionCreateRequestSchema,
  executionCreateResponseSchema,
  confirmationDecisionRequestSchema,
  confirmationDecisionResponseSchema,
  envelopeSchema
].forEach((schema) => ajv.addSchema(schema));

const executionCreateRequestValidator = ajv.getSchema("goyais.protocol.v2.execution-create.request")!;
const executionCreateResponseValidator = ajv.getSchema("goyais.protocol.v2.execution-create.response")!;
const eventEnvelopeValidator = ajv.getSchema("goyais.protocol.v2.event-envelope")!;
const confirmationDecisionRequestValidator = ajv.getSchema("goyais.protocol.v2.confirmation-decision.request")!;
const confirmationDecisionResponseValidator = ajv.getSchema("goyais.protocol.v2.confirmation-decision.response")!;

export function validateExecutionCreateRequest(payload: unknown): boolean {
  return !!executionCreateRequestValidator(payload);
}

export function validateExecutionCreateResponse(payload: unknown): boolean {
  return !!executionCreateResponseValidator(payload);
}

export function validateEventEnvelope(payload: unknown): boolean {
  return !!eventEnvelopeValidator(payload);
}

export function validateConfirmationDecisionRequest(payload: unknown): boolean {
  return !!confirmationDecisionRequestValidator(payload);
}

export function validateConfirmationDecisionResponse(payload: unknown): boolean {
  return !!confirmationDecisionResponseValidator(payload);
}

export function getValidationErrors(): string[] {
  const errors = [
    executionCreateRequestValidator.errors,
    executionCreateResponseValidator.errors,
    eventEnvelopeValidator.errors,
    confirmationDecisionRequestValidator.errors,
    confirmationDecisionResponseValidator.errors
  ]
    .flat()
    .filter((error): error is NonNullable<typeof error> => Boolean(error));

  return errors.map((error) => `${error.instancePath} ${error.message}`.trim());
}
