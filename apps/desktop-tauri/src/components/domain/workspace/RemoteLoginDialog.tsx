import { FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useSettingsStore } from "@/stores/settingsStore";

interface RemoteLoginDialogProps {
  open: boolean;
  loading: boolean;
  errorMessage?: string;
  onOpenChange: (open: boolean) => void;
  onSubmit: (payload: { serverUrl: string; email: string; password: string }) => Promise<void>;
}

export function RemoteLoginDialog({
  open,
  loading,
  errorMessage,
  onOpenChange,
  onSubmit
}: RemoteLoginDialogProps) {
  const { t } = useTranslation();
  const defaultServerUrl = useSettingsStore((state) => state.localProcessConfig.connections.defaultRemoteServerUrl);
  const [serverUrl, setServerUrl] = useState(defaultServerUrl);
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("");

  useEffect(() => {
    if (!open) {
      setPassword("");
    }
    if (open && defaultServerUrl.trim()) {
      setServerUrl(defaultServerUrl);
    }
  }, [defaultServerUrl, open]);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    await onSubmit({
      serverUrl: serverUrl.trim(),
      email: email.trim(),
      password
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("workspace.remoteLogin.title")}</DialogTitle>
          <DialogDescription>{t("workspace.remoteLogin.description")}</DialogDescription>
        </DialogHeader>

        <form className="space-y-3" onSubmit={(event) => void handleSubmit(event)}>
          <label className="grid gap-1 text-small text-muted-foreground">
            {t("workspace.remoteLogin.serverUrl")}
            <Input
              value={serverUrl}
              onChange={(event) => setServerUrl(event.target.value)}
              required
            />
          </label>

          <label className="grid gap-1 text-small text-muted-foreground">
            {t("workspace.remoteLogin.email")}
            <Input type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
          </label>

          <label className="grid gap-1 text-small text-muted-foreground">
            {t("workspace.remoteLogin.password")}
            <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
          </label>

          {errorMessage ? <p className="text-small text-destructive">{errorMessage}</p> : null}

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("workspace.cancel")}
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? t("workspace.loading") : t("workspace.remoteLogin.submit")}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
