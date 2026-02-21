export interface MenuRecord {
  menu_id: string;
  parent_id: string | null;
  sort_order: number;
  route: string | null;
  icon_key: string | null;
  i18n_key: string;
}

export interface NavigationMenuNode {
  menu_id: string;
  route: string | null;
  icon_key: string | null;
  i18n_key: string;
  children: NavigationMenuNode[];
}

export function buildMenuTree(records: MenuRecord[]): NavigationMenuNode[] {
  const sorted = [...records].sort((a, b) => {
    if (a.sort_order === b.sort_order) {
      return a.menu_id.localeCompare(b.menu_id);
    }
    return a.sort_order - b.sort_order;
  });

  const byId = new Map<string, NavigationMenuNode>();
  for (const menu of sorted) {
    byId.set(menu.menu_id, {
      menu_id: menu.menu_id,
      route: menu.route,
      icon_key: menu.icon_key,
      i18n_key: menu.i18n_key,
      children: []
    });
  }

  const roots: NavigationMenuNode[] = [];
  for (const menu of sorted) {
    const node = byId.get(menu.menu_id);
    if (!node) {
      continue;
    }

    if (menu.parent_id && byId.has(menu.parent_id)) {
      byId.get(menu.parent_id)!.children.push(node);
      continue;
    }

    roots.push(node);
  }

  return roots;
}
