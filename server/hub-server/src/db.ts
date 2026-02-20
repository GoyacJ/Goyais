import { randomUUID } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { DatabaseSync, type SQLInputValue } from "node:sqlite";

import { HubServerError } from "./errors";
import type { MenuRecord } from "./services/navigation";

export interface UserSummary {
  user_id: string;
  email: string;
  display_name: string;
}

export interface WorkspaceSummary {
  workspace_id: string;
  name: string;
  slug: string;
}

interface CreateBootstrapAdminInput {
  email: string;
  passwordHash: string;
  displayName: string;
  tokenId: string;
  tokenHash: string;
  tokenCreatedAt: string;
  tokenExpiresAt: string;
}

export interface UserAuthRecord {
  user_id: string;
  email: string;
  password_hash: string;
  display_name: string;
  status: "active" | "disabled";
}

export interface AuthTokenRecord {
  token_id: string;
  user_id: string;
  expires_at: string;
  email: string;
  display_name: string;
  user_status: "active" | "disabled";
}

export interface MembershipSummary {
  workspace_id: string;
  workspace_name: string;
  workspace_slug: string;
  role_name: string;
}

export interface WorkspaceMembershipSummary {
  workspace_id: string;
  name: string;
  slug: string;
  role_name: string;
}

export interface MembershipRole {
  role_id: string;
  role_name: string;
}

export interface ProjectRecord {
  project_id: string;
  workspace_id: string;
  name: string;
  root_uri: string;
  created_at: string;
  updated_at: string;
}

export interface ModelConfigRecord {
  model_config_id: string;
  workspace_id: string;
  provider: string;
  model: string;
  base_url: string | null;
  temperature: number;
  max_tokens: number | null;
  secret_ref: string;
  created_at: string;
  updated_at: string;
}

export interface SecretRecord {
  secret_ref: string;
  workspace_id: string;
  kind: "api_key";
  value_encrypted: string;
  created_by: string;
  created_at: string;
}

export class HubDatabase {
  private readonly db: DatabaseSync;

  constructor(filePath: string) {
    fs.mkdirSync(path.dirname(filePath), { recursive: true });
    this.db = new DatabaseSync(filePath);
    this.db.exec("PRAGMA journal_mode = WAL;");
    this.db.exec("PRAGMA foreign_keys = ON;");
  }

  migrate(migrationsDir: string): void {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS schema_migrations (
        version TEXT PRIMARY KEY,
        applied_at TEXT NOT NULL
      );
    `);

    const files = fs
      .readdirSync(migrationsDir)
      .filter((file) => file.endsWith(".sql"))
      .sort();

    for (const file of files) {
      const exists = this.db.prepare("SELECT 1 FROM schema_migrations WHERE version = ?").get(file);
      if (exists) {
        continue;
      }

      const sql = fs.readFileSync(path.join(migrationsDir, file), "utf8");
      this.db.exec(sql);
      this.db
        .prepare(
          "INSERT INTO schema_migrations(version, applied_at) VALUES(?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))"
        )
        .run(file);
    }
  }

  scalar<T extends number | string>(sql: string, ...params: SQLInputValue[]): T {
    const row = this.db.prepare(sql).get(...params) as Record<string, T>;
    return row[Object.keys(row)[0]];
  }

  getSetupStatus(): { setupMode: boolean; usersCount: number; setupCompleted: number } {
    const usersCount = Number(this.scalar<number>("SELECT COUNT(*) AS total FROM users"));
    const setupCompleted = Number(
      this.scalar<number>("SELECT setup_completed FROM system_state WHERE singleton_id = 1")
    );

    return {
      setupMode: usersCount === 0 || setupCompleted === 0,
      usersCount,
      setupCompleted
    };
  }

  createBootstrapAdmin(input: CreateBootstrapAdminInput): { user: UserSummary; workspace: WorkspaceSummary } {
    const setup = this.getSetupStatus();
    if (!setup.setupMode) {
      throw new HubServerError({
        code: "E_SETUP_COMPLETED",
        message: "Bootstrap has already been completed.",
        retryable: false,
        statusCode: 409,
        causeType: "setup_completed"
      });
    }

    const userId = randomUUID();
    const workspaceId = randomUUID();
    const ownerRoleId = randomUUID();
    const memberRoleId = randomUUID();
    const now = new Date().toISOString();

    try {
      this.db.exec("BEGIN IMMEDIATE");

      this.db
        .prepare(
          `
          INSERT INTO users(user_id, email, password_hash, display_name, status, created_at)
          VALUES(?, ?, ?, ?, 'active', ?)
        `
        )
        .run(userId, input.email, input.passwordHash, input.displayName, now);

      this.db
        .prepare(
          `
          INSERT INTO workspaces(workspace_id, name, slug, created_by, created_at)
          VALUES(?, 'Default', 'default', ?, ?)
        `
        )
        .run(workspaceId, userId, now);

      this.db
        .prepare(
          `
          INSERT INTO roles(role_id, workspace_id, name, is_system, created_at)
          VALUES(?, ?, 'Owner', 1, ?)
        `
        )
        .run(ownerRoleId, workspaceId, now);

      this.db
        .prepare(
          `
          INSERT INTO roles(role_id, workspace_id, name, is_system, created_at)
          VALUES(?, ?, 'Member', 1, ?)
        `
        )
        .run(memberRoleId, workspaceId, now);

      this.db
        .prepare(
          `
          INSERT OR IGNORE INTO role_permissions(role_id, perm_key)
          SELECT ?, perm_key FROM permissions
        `
        )
        .run(ownerRoleId);

      this.db
        .prepare(
          `
          INSERT OR IGNORE INTO role_menus(role_id, menu_id)
          SELECT ?, menu_id FROM menus
        `
        )
        .run(ownerRoleId);

      const memberPermissions = ["workspace:read", "project:read", "run:create", "modelconfig:read"];
      for (const permKey of memberPermissions) {
        this.db
          .prepare(
            `
            INSERT OR IGNORE INTO role_permissions(role_id, perm_key)
            VALUES(?, ?)
          `
          )
          .run(memberRoleId, permKey);
      }

      const memberMenus = ["nav_projects", "nav_run", "nav_models"];
      for (const menuId of memberMenus) {
        this.db
          .prepare(
            `
            INSERT OR IGNORE INTO role_menus(role_id, menu_id)
            VALUES(?, ?)
          `
          )
          .run(memberRoleId, menuId);
      }

      this.db
        .prepare(
          `
          INSERT INTO workspace_members(workspace_id, user_id, role_id, status, joined_at)
          VALUES(?, ?, ?, 'active', ?)
        `
        )
        .run(workspaceId, userId, ownerRoleId, now);

      this.db
        .prepare(
          `
          INSERT INTO auth_tokens(token_id, token_hash, user_id, expires_at, created_at)
          VALUES(?, ?, ?, ?, ?)
        `
        )
        .run(input.tokenId, input.tokenHash, userId, input.tokenExpiresAt, input.tokenCreatedAt);

      this.db
        .prepare(
          `
          UPDATE system_state
          SET setup_completed = 1, updated_at = ?
          WHERE singleton_id = 1
        `
        )
        .run(now);

      this.db.exec("COMMIT");

      return {
        user: {
          user_id: userId,
          email: input.email,
          display_name: input.displayName
        },
        workspace: {
          workspace_id: workspaceId,
          name: "Default",
          slug: "default"
        }
      };
    } catch (error) {
      this.db.exec("ROLLBACK");

      if (error instanceof HubServerError) {
        throw error;
      }

      if (error instanceof Error && error.message.includes("UNIQUE constraint failed: users.email")) {
        throw new HubServerError({
          code: "E_BOOTSTRAP_CONFLICT",
          message: "Admin user already exists.",
          retryable: false,
          statusCode: 409,
          causeType: "email_exists"
        });
      }

      throw error;
    }
  }

  getUserByEmail(email: string): UserAuthRecord | null {
    const row = this.db
      .prepare(
        `
        SELECT user_id, email, password_hash, display_name, status
        FROM users
        WHERE email = ?
      `
      )
      .get(email) as UserAuthRecord | undefined;

    return row ?? null;
  }

  createAuthToken(params: {
    tokenId: string;
    tokenHash: string;
    userId: string;
    expiresAt: string;
    createdAt: string;
  }): void {
    this.db
      .prepare(
        `
        INSERT INTO auth_tokens(token_id, token_hash, user_id, expires_at, created_at)
        VALUES(?, ?, ?, ?, ?)
      `
      )
      .run(params.tokenId, params.tokenHash, params.userId, params.expiresAt, params.createdAt);
  }

  getAuthTokenByHash(tokenHash: string): AuthTokenRecord | null {
    const row = this.db
      .prepare(
        `
        SELECT
          t.token_id,
          t.user_id,
          t.expires_at,
          u.email,
          u.display_name,
          u.status AS user_status
        FROM auth_tokens t
        JOIN users u ON u.user_id = t.user_id
        WHERE t.token_hash = ?
      `
      )
      .get(tokenHash) as AuthTokenRecord | undefined;

    return row ?? null;
  }

  touchAuthToken(tokenId: string): void {
    this.db
      .prepare(
        `
        UPDATE auth_tokens
        SET last_used_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
        WHERE token_id = ?
      `
      )
      .run(tokenId);
  }

  listMemberships(userId: string): MembershipSummary[] {
    const rows = this.db
      .prepare(
        `
        SELECT
          wm.workspace_id,
          w.name AS workspace_name,
          w.slug AS workspace_slug,
          r.name AS role_name
        FROM workspace_members wm
        JOIN workspaces w ON w.workspace_id = wm.workspace_id
        JOIN roles r ON r.role_id = wm.role_id
        WHERE wm.user_id = ?
          AND wm.status = 'active'
        ORDER BY w.created_at ASC
      `
      )
      .all(userId) as unknown as MembershipSummary[];

    return rows;
  }

  listWorkspacesForUser(userId: string): WorkspaceMembershipSummary[] {
    const rows = this.db
      .prepare(
        `
        SELECT
          wm.workspace_id,
          w.name,
          w.slug,
          r.name AS role_name
        FROM workspace_members wm
        JOIN workspaces w ON w.workspace_id = wm.workspace_id
        JOIN roles r ON r.role_id = wm.role_id
        WHERE wm.user_id = ?
          AND wm.status = 'active'
        ORDER BY w.created_at ASC
      `
      )
      .all(userId) as unknown as WorkspaceMembershipSummary[];

    return rows;
  }

  getMembershipRole(userId: string, workspaceId: string): MembershipRole | null {
    const row = this.db
      .prepare(
        `
        SELECT
          r.role_id,
          r.name AS role_name
        FROM workspace_members wm
        JOIN roles r ON r.role_id = wm.role_id
        WHERE wm.user_id = ?
          AND wm.workspace_id = ?
          AND wm.status = 'active'
        LIMIT 1
      `
      )
      .get(userId, workspaceId) as MembershipRole | undefined;

    return row ?? null;
  }

  listPermissionsForRole(roleId: string): string[] {
    const rows = this.db
      .prepare(
        `
        SELECT perm_key
        FROM role_permissions
        WHERE role_id = ?
        ORDER BY perm_key ASC
      `
      )
      .all(roleId) as Array<{ perm_key: string }>;

    return rows.map((row) => row.perm_key);
  }

  listMenusForRole(roleId: string): MenuRecord[] {
    const rows = this.db
      .prepare(
        `
        SELECT
          m.menu_id,
          m.parent_id,
          m.sort_order,
          m.route,
          m.icon_key,
          m.i18n_key
        FROM role_menus rm
        JOIN menus m ON m.menu_id = rm.menu_id
        WHERE rm.role_id = ?
        ORDER BY m.sort_order ASC, m.menu_id ASC
      `
      )
      .all(roleId) as unknown as MenuRecord[];

    return rows;
  }
}
