import { Loader2, Plus, Trash2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  createSkillSet,
  deleteSkillSet,
  type HubSkillSet,
  listSkillSets,
  updateSkillSet
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

function canManageSkills(kind: "local" | "remote", permissions: string[]): boolean {
  if (kind === "local") return true;
  return permissions.includes("skill:write");
}

export function SkillsPanel() {
  const { t } = useTranslation();
  const { addToast } = useToast();

  const profile = useWorkspaceStore(selectCurrentProfile);
  const permissions = useWorkspaceStore(selectCurrentPermissions);
  const writable = canManageSkills(profile?.kind ?? "local", permissions);

  const [loading, setLoading] = useState(false);
  const [skillSets, setSkillSets] = useState<HubSkillSet[]>([]);

  const [addOpen, setAddOpen] = useState(false);
  const [addName, setAddName] = useState("");
  const [addDesc, setAddDesc] = useState("");
  const [addSaving, setAddSaving] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editDesc, setEditDesc] = useState("");
  const [editSaving, setEditSaving] = useState(false);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const ctx = await resolveHubContext(profile);
      const res = await listSkillSets(ctx.serverUrl, ctx.token, ctx.workspaceId);
      setSkillSets(res.skill_sets ?? []);
    } catch (err) {
      addToast({ title: t("skills.loadFailed"), description: (err as Error).message, variant: "error" });
    } finally {
      setLoading(false);
    }
  }, [addToast, profile, t]);

  useEffect(() => { void refresh(); }, [refresh]);

  const onAdd = useCallback(async () => {
    if (!addName.trim()) return;
    setAddSaving(true);
    try {
      const ctx = await resolveHubContext(profile);
      await createSkillSet(ctx.serverUrl, ctx.token, ctx.workspaceId, {
        name: addName.trim(),
        description: addDesc.trim() || undefined
      });
      await refresh();
      setAddOpen(false);
      setAddName("");
      setAddDesc("");
      addToast({ title: t("skills.createSuccess"), variant: "success" });
    } catch (err) {
      addToast({ title: t("skills.createFailed"), description: (err as Error).message, variant: "error" });
    } finally {
      setAddSaving(false);
    }
  }, [addDesc, addName, addToast, profile, refresh, t]);

  const onEdit = useCallback((item: HubSkillSet) => {
    setEditingId(item.skill_set_id);
    setEditName(item.name);
    setEditDesc(item.description ?? "");
    setEditOpen(true);
  }, []);

  const onSaveEdit = useCallback(async () => {
    if (!editingId) return;
    setEditSaving(true);
    try {
      const ctx = await resolveHubContext(profile);
      await updateSkillSet(ctx.serverUrl, ctx.token, ctx.workspaceId, editingId, {
        name: editName.trim() || undefined,
        description: editDesc.trim() || undefined
      });
      await refresh();
      setEditOpen(false);
      addToast({ title: t("skills.updateSuccess"), variant: "success" });
    } catch (err) {
      addToast({ title: t("skills.updateFailed"), description: (err as Error).message, variant: "error" });
    } finally {
      setEditSaving(false);
    }
  }, [addToast, editDesc, editingId, editName, profile, refresh, t]);

  const onDelete = useCallback(async (id: string) => {
    try {
      const ctx = await resolveHubContext(profile);
      await deleteSkillSet(ctx.serverUrl, ctx.token, ctx.workspaceId, id);
      await refresh();
      addToast({ title: t("skills.deleteSuccess"), variant: "success" });
    } catch (err) {
      addToast({ title: t("skills.deleteFailed"), description: (err as Error).message, variant: "error" });
    }
  }, [addToast, profile, refresh, t]);

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-3 pb-3">
          <div>
            <CardTitle>{t("skills.title")}</CardTitle>
            <CardDescription>{t("skills.description")}</CardDescription>
          </div>
          <Button type="button" size="sm" disabled={!writable} onClick={() => setAddOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("skills.add")}
          </Button>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center gap-2 text-small text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>{t("skills.loading")}</span>
            </div>
          ) : skillSets.length === 0 ? (
            <p className="text-small text-muted-foreground">{t("skills.empty")}</p>
          ) : (
            <div className="overflow-auto rounded-panel border border-border-subtle bg-background/40 scrollbar-subtle">
              <table className="w-full text-small text-foreground">
                <thead className="bg-muted/55 text-xs text-muted-foreground">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium">{t("skills.colName")}</th>
                    <th className="px-3 py-2 text-left font-medium">{t("skills.colDescription")}</th>
                    <th className="px-3 py-2 text-right font-medium">{t("models.actions")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border-subtle">
                  {skillSets.map((item) => (
                    <tr key={item.skill_set_id} className="align-middle">
                      <td className="px-3 py-2 font-medium">{item.name}</td>
                      <td className="max-w-[24rem] truncate px-3 py-2 text-muted-foreground">
                        {item.description ?? "—"}
                      </td>
                      <td className="px-3 py-2">
                        <div className="flex justify-end gap-2">
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
                            onClick={() => void onDelete(item.skill_set_id)}
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

      <Dialog
        open={addOpen}
        onOpenChange={(open) => {
          setAddOpen(open);
          if (!open) { setAddName(""); setAddDesc(""); }
        }}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("skills.add")}</DialogTitle>
            <DialogDescription>{t("skills.addDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <p className="text-small text-muted-foreground">{t("skills.colName")}</p>
              <Input
                className="h-9"
                value={addName}
                onChange={(e) => setAddName(e.target.value)}
                placeholder={t("skills.namePlaceholder")}
              />
            </div>
            <div className="space-y-1">
              <p className="text-small text-muted-foreground">{t("skills.colDescription")}</p>
              <Input
                className="h-9"
                value={addDesc}
                onChange={(e) => setAddDesc(e.target.value)}
                placeholder={t("skills.descriptionPlaceholder")}
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setAddOpen(false)}>
              {t("workspace.cancel")}
            </Button>
            <Button type="button" disabled={!addName.trim() || addSaving} onClick={() => void onAdd()}>
              {addSaving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              {t("skills.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("skills.edit")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <p className="text-small text-muted-foreground">{t("skills.colName")}</p>
              <Input className="h-9" value={editName} onChange={(e) => setEditName(e.target.value)} />
            </div>
            <div className="space-y-1">
              <p className="text-small text-muted-foreground">{t("skills.colDescription")}</p>
              <Input className="h-9" value={editDesc} onChange={(e) => setEditDesc(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditOpen(false)}>
              {t("workspace.cancel")}
            </Button>
            <Button type="button" disabled={editSaving} onClick={() => void onSaveEdit()}>
              {editSaving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              {t("skills.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
