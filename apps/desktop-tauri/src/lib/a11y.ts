export function isActivationKey(event: KeyboardEvent | React.KeyboardEvent, key: "enter" | "escape") {
  if (key === "enter") {
    return event.key === "Enter";
  }
  return event.key === "Escape";
}

export function isMacPlatform() {
  return typeof navigator !== "undefined" && /Mac|iPhone|iPad|iPod/.test(navigator.platform);
}

export function primaryShortcutLabel(key: string) {
  return `${isMacPlatform() ? "Cmd" : "Ctrl"}+${key.toUpperCase()}`;
}
