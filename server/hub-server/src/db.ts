import { randomUUID } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { DatabaseSync, type SQLInputValue } from "node:sqlite";

import { HubServerError } from "./errors";

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
}
