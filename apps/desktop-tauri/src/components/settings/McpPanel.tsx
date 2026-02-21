import { Loader2, Plus, Trash2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  createMCPConnector,
  deleteMCPConnector,
  type HubMCPConnector,
  listMCPConnectors,
  updateMCPConnector
} from "@/api/hubClient";
import { resolveHubContext } from "@/api/sessionDataSource";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { selectCurrentPermissions, selectCurrentProfile, useWorkspaceStore } from "@/stores/workspaceStore";

const TRANSPORT_OPTIONS = ["stdio", "sse", "streamable_http"] as const;
type Transport = (typeof TRANSPORT_OPTIONS)[number];

function canManageMCP(kind: "local" | "remote", permissions: string[]): boolean {
  if (kind === "local") return true;
  return permissions.includes("mcp:write");
}

interface ConnectorDraft {
  name: string;
  transport: Transport;
  endpoint: string;
  secret_ref: string;
}

const DEFAULT_DRAFT: ConnectorDraft = { name: "", transport: "sse", endpoint: "", secret_ref: "" };

export function McpPanel() {
  const { t } = useTranslation();
  const { addToast } = useToast();

  const profile = useWorkspaceStore(selectCurrentProfile);
  const permissions = useWorkspaceStore(selectCurrentPermissions);
  const writable = canManageMCP(profile?.kind ?? "local", permissions);

  const [loading, setLoading] = useState(false);
  const [connectors, setConnectors] = useState<HubMCPConnector[]>([]);

  const [addOpen, setAddOpen] = useState(false);
  const [addDraft, setAddDraft] = useState<ConnectorDraft>(DEFAULT_DRAFT);
  const [addSaving, setAddSaving] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editDraft, setEditDraft] = useState<ConnectorDraft>(DEFAULT_DRAFT);
  const [editSaving, setEditSaving] = useState(false);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const ctx = await resolveHubContext(profile);
      const res = await listMCPConnectors(ctx.serverUrl, ctx.token, ctx.workspaceId);
      setConnectors(res.mcp_connectors ?? []);
    } catch (err) {
      addToast({ title: t("mcp.loadFailed"), description: (err as Error).message, variant: "error" });
    } finally {
      setLoading(false);
    }
  }, [addToast, profile, t]);

  useEffect(() => { void refresh(); }, [refresh]);

  const onAdd = useCallback(async () => {
    if (!addDraft.name.trim() || !addDraft.endpoint.trim()) return;
    setAddSaving(true);
    try {
      const ctx = await resolveHubContext(profile);
      await createMCPConnector(ctx.serverUrl, ctx.token, ctx.workspaceId, {
        name: addDraft.name.trim(),
        transport: addDraft.transport,
        endpoint: addDraft.endpoint.trim(),
        secret_ref: addDraft.secret_ref.trim() || undefined
      });
      await refresh();
      setAddOpen(false);
      setAddDraft(DEFAULT_DRAFT);
      addToast({ title: t("mcp.createSuccess"), variant: "success" });
    } catch (err) {
      addToast({ title: t("mcp.createFailed"), description: (err as Error).message, variant: "error" });
    } finally {
      setAddSaving(false);
    }
  }, [addDraft, addToast, profile, refresh, t]);

  const onEdit = useCallback((item: HubMCPConnector) => {
    setEditingId(item.connector_id);
    setEditDraft({
      name: item.name,
      transport: item.transport,
      endpoint: item.endpoint,
      secret_ref: item.secret_ref ?? ""
    });
    setEditOpen(true);
  }, []);

  const onSaveEdit = useCallback(async () => {
    if (!editingId) return;
    setEditSaving(true);
    try {
      const ctx = await resolveHubContext(profile);
      await updateMCPConnector(ctx.serverUrl, ctx.token, ctx.workspaceId, editingId, {
        name: editDraft.name.trim() || undefined,
        transport: editDraft.transport,
        endpoint: editDraft.endpoint.trim() || undefined,
        secret_ref: editDraft.secret_ref.trim() || undefined
      });
      await refresh();
      setEditOpen(false);
      addToast({ title: t("mcp.updateSuccess"), variant: "success" });
    } catch (err) {
      addToast({ title: t("mcp.updateFailed"), description: (err as Error).message, variant: "error" });
    } finally {
      setEditSaving(false);
    }
  }, [addToast, editDraft, editingId, profile, refresh, t]);

  const onDelete = useCallback(async (id: string) => {
    try {
      const ctx = await resolveHubContext(profile);
      await deleteMCPConnector(ctx.serverUrl, ctx.token, ctx.workspaceId, id);
      await refresh();
      addToast({ title: t("mcp.deleteSuccess"), variant: "success" });
    } catch (err) {
      addToast({ title: t("mcp.deleteFailed"), description: (err as Error).message, variant: "error" });
    }
  }, [addToast, profile, refresh, t]);

  const onToggleEnabled = useCallback(async (item: HubMCPConnector) => {
    try {
      const ctx = await resolveHubContext(profile);
      await updateMCPConnector(ctx.serverUrl, ctx.token, ctx.workspaceId, item.connector_id, {
        enabled: !item.enabled
      });
      await refresh();
    } catch (err) {
      addToast({ title: t("mcp.updateFailed"), description: (err as Error).message, variant: "error" });
    }
  }, [addToast, profile, refresh, t]);

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-3 pb-3">
          <div>
            <CardTitle>{t("mcp.title")}</CardTitle>
            <CardDescription>{t("mcp.description")}</CardDescription>
          </div>
          <Button type="button" size="sm" disabled={!writable} onClick={() => setAddOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("mcp.add")}
          </Button>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center gap-2 text-small text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>{t("mcp.loading")}</span>
            </div>
          ) : connectors.length === 0 ? (
            <p className="text-small text-muted-foreground">{t("mcp.empty")}</p>
          ) : (
            <div className="overflow-auto rounded-panel border border-border-subtle bg-background/40 scrollbar-subtle">
              <table className="w-full min-w-[640px] text-small text-foreground">
                <thead className="bg-muted/55 text-xs text-muted-foreground">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium">{t("mcp.colName")}</th>
                    <th className="px-3 py-2 text-left font-medium">{t("mcp.colTransport")}</th>
                    <th className="px-3 py-2 text-left font-medium">{t("mcp.colEndpoint")}</th>
                    <th className="px-3 py-2 text-left font-medium">{t("mcp.colStatus")}</th>
                    <th className="px-3 py-2 text-right font-medium">{t("models.actions")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border-subtle">
                  {connectors.map((item) => (
                    <tr key={item.connector_id} className="align-middle">
                      <td className="px-3 py-2 font-medium">{item.name}</td>
                      <td className="px-3 py-2 text-muted-foreground">{item.transport}</td>
                      <td className="max-w-[20rem] truncate px-3 py-2 text-muted-foreground" title={item.endpoint}>
                        {item.endpoint}
                      </td>
                      <td className="px-3 py-2">
                        <span className={item.enabled ? "text-success" : "text-muted-foreground"}>
                          {item.enabled ? t("mcp.enabled") : t("mcp.disabled")}
                        </span>
                      </td>
                      <td className="px-3 py-2">
                        <div className="flex justify-end gap-2">
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            disabled={!writable}
                            onClick={() => void onToggleEnabled(item)}
                          >
                            {item.enabled ? t("mcp.disable") : t("mcp.enable")}
                          </Button>
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            disabled={!writable}
                            onClick={() => onEdit(item)}
                          >
                            {t("models.edit")}
                          </Button>
                          <Button
                            type="button"
                            size="sm"
                            variant="destructive"
                            disabled={!writable}
                            onClick={() => void onDelete(item.connector_id)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            {t("models.delete")}
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Add dialog */}
      <Dialog
        open={addOpen}
        onOpenChange={(open) => {
          setAddOpen(open);
          if (!open) setAddDraft(DEFAULT_DRAFT);
        }}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("mcp.add")}</DialogTitle>
            <DialogDescription>{t("mcp.addDescription")}</DialogDescription>
          </DialogHeader>
          <ConnectorForm draft={addDraft} onChange={setAddDraft} t={t} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setAddOpen(false)}>
              {t("workspace.cancel")}
            </Button>
            <Button
              type="button"
              disabled={!addDraft.name.trim() || !addDraft.endpoint.trim() || addSaving}
              onClick={() => void onAdd()}
            >
              {addSaving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              {t("mcp.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("mcp.edit")}</DialogTitle>
          </DialogHeader>
          <ConnectorForm draft={editDraft} onChange={setEditDraft} t={t} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditOpen(false)}>
              {t("workspace.cancel")}
            </Button>
            <Button type="button" disabled={editSaving} onClick={() => void onSaveEdit()}>
              {editSaving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              {t("mcp.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function ConnectorForm({
  draft,
  onChange,
  t
}: {
  draft: ConnectorDraft;
  onChange: (d: ConnectorDraft) => void;
  t: (key: string) => string;
}) {
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <p className="text-small text-muted-foreground">{t("mcp.colName")}</p>
        <Input
          className="h-9"
          value={draft.name}
          onChange={(e) => onChange({ ...draft, name: e.target.value })}
          placeholder={t("mcp.namePlaceholder")}
        />
      </div>
      <div className="space-y-1">
        <p className="text-small text-muted-foreground">{t("mcp.colTransport")}</p>
        <select
          className="h-9 w-full rounded-control border border-border bg-background px-2 text-small text-foreground"
          value={draft.transport}
          onChange={(e) => onChange({ ...draft, transport: e.target.value as Transport })}
        >
          {TRANSPORT_OPTIONS.map((opt) => (
            <option key={opt} value={opt}>{opt}</option>
          ))}
        </select>
      </div>
      <div className="space-y-1">
        <p className="text-small text-muted-foreground">{t("mcp.colEndpoint")}</p>
        <Input
          className="h-9"
          value={draft.endpoint}
          onChange={(e) => onChange({ ...draft, endpoint: e.target.value })}
          placeholder={t("mcp.endpointPlaceholder")}
        />
      </div>
      <div className="space-y-1">
        <p className="text-small text-muted-foreground">{t("mcp.secretRef")}</p>
        <Input
          className="h-9"
          value={draft.secret_ref}
          onChange={(e) => onChange({ ...draft, secret_ref: e.target.value })}
          placeholder={t("mcp.secretRefPlaceholder")}
        />
        <p className="text-xs text-muted-foreground">{t("mcp.secretRefHelp")}</p>
      </div>
    </div>
  );
}
