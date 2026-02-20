import type { FastifyRequest } from "fastify";

export function assertToken(request: FastifyRequest, expectedToken: string): void {
  const header = request.headers.authorization;
  if (!header || !header.startsWith("Bearer ")) {
    throw new Error("missing bearer token");
  }

  const token = header.replace(/^Bearer\s+/, "").trim();
  if (token !== expectedToken) {
    throw new Error("invalid bearer token");
  }
}
