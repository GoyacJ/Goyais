import { computed } from "vue";

import { useI18n } from "@/shared/i18n";
import { getMenuKeysForContext, getMenuSchemaMap, type MenuContext } from "@/shared/navigation/menuSchema";
import { getMenuVisibility } from "@/shared/stores/navigationStore";
import type { MenuKey, PermissionVisibility } from "@/shared/types/api";

export type MenuEntry = {
  key: MenuKey;
  label: string;
  path: string;
  visibility: PermissionVisibility;
};

export function useRemoteConfigMenu() {
  return useMenuEntries("account");
}

export function useLocalSettingsMenu() {
  return useMenuEntries("settings");
}

function useMenuEntries(context: MenuContext) {
  const { t } = useI18n();
  const schemaMap = getMenuSchemaMap();

  return computed<MenuEntry[]>(() =>
    getMenuKeysForContext(context)
      .map((key) => {
        const schema = schemaMap.get(key);
        if (!schema) {
          return undefined;
        }

        return createEntry(key, t(schema.labelKey), schema.path);
      })
      .filter((entry): entry is MenuEntry => entry !== undefined)
  );
}

function createEntry(key: MenuKey, label: string, path: string): MenuEntry {
  return {
    key,
    label,
    path,
    visibility: getMenuVisibility(key)
  };
}
