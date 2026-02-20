export function isMacPlatform() {
  return typeof navigator !== "undefined" && /Mac|iPhone|iPad|iPod/.test(navigator.platform);
}

export function isPrimaryModifier(event: KeyboardEvent) {
  return isMacPlatform() ? event.metaKey : event.ctrlKey;
}

export function isPaletteShortcut(event: KeyboardEvent) {
  return isPrimaryModifier(event) && !event.shiftKey && event.key.toLowerCase() === "k";
}

export function isEditableElement(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tag = target.tagName.toLowerCase();
  return tag === "input" || tag === "textarea" || tag === "select";
}

export function primaryShortcutLabel(key: string) {
  return `${isMacPlatform() ? "Cmd" : "Ctrl"}+${key.toUpperCase()}`;
}
