import Ajv from "ajv";
import addFormats from "ajv-formats";
import envelopeSchema from "../schemas/v2/event-envelope.schema.json";
import runCreateRequestSchema from "../schemas/v2/run-create.request.schema.json";
import runCreateResponseSchema from "../schemas/v2/run-create.response.schema.json";
import toolConfirmationRequestSchema from "../schemas/v2/tool-confirmation.request.schema.json";
import toolConfirmationResponseSchema from "../schemas/v2/tool-confirmation.response.schema.json";
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
  runCreateRequestSchema,
  runCreateResponseSchema,
  toolConfirmationRequestSchema,
  toolConfirmationResponseSchema,
  envelopeSchema
].forEach((schema) => ajv.addSchema(schema));

const runCreateRequestValidator = ajv.getSchema("goyais.protocol.v2.run-create.request")!;
const runCreateResponseValidator = ajv.getSchema("goyais.protocol.v2.run-create.response")!;
const eventEnvelopeValidator = ajv.getSchema("goyais.protocol.v2.event-envelope")!;
const toolConfirmationRequestValidator = ajv.getSchema("goyais.protocol.v2.tool-confirmation.request")!;
const toolConfirmationResponseValidator = ajv.getSchema("goyais.protocol.v2.tool-confirmation.response")!;

export function validateRunCreateRequest(payload: unknown): boolean {
  return !!runCreateRequestValidator(payload);
}

export function validateRunCreateResponse(payload: unknown): boolean {
  return !!runCreateResponseValidator(payload);
}

export function validateEventEnvelope(payload: unknown): boolean {
  return !!eventEnvelopeValidator(payload);
}

export function validateToolConfirmationRequest(payload: unknown): boolean {
  return !!toolConfirmationRequestValidator(payload);
}

export function validateToolConfirmationResponse(payload: unknown): boolean {
  return !!toolConfirmationResponseValidator(payload);
}

export function getValidationErrors(): string[] {
  const errors = [
    runCreateRequestValidator.errors,
    runCreateResponseValidator.errors,
    eventEnvelopeValidator.errors,
    toolConfirmationRequestValidator.errors,
    toolConfirmationResponseValidator.errors
  ]
    .flat()
    .filter((error): error is NonNullable<typeof error> => Boolean(error));

  return errors.map((error) => `${error.instancePath} ${error.message}`.trim());
}
