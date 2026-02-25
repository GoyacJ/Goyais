export function createMockId(prefix: string): string {
  const randomPart = Math.random().toString(16).slice(2, 8);
  return `${prefix}_${randomPart}`;
}
