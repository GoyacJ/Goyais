import { FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

interface SetupAdminDialogProps {
  open: boolean;
  serverUrl: string;
  loading: boolean;
  errorMessage?: string;
  onOpenChange: (open: boolean) => void;
  onSubmit: (payload: {
    bootstrapToken: string;
    email: string;
    password: string;
    displayName: string;
  }) => Promise<void>;
}

export function SetupAdminDialog({
  open,
  serverUrl,
  loading,
  errorMessage,
  onOpenChange,
  onSubmit
}: SetupAdminDialogProps) {
  const { t } = useTranslation();
  const [bootstrapToken, setBootstrapToken] = useState("");
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("Admin");

  useEffect(() => {
    if (!open) {
      setPassword("");
      setBootstrapToken("");
    }
  }, [open]);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    await onSubmit({
      bootstrapToken: bootstrapToken.trim(),
      email: email.trim(),
      password,
      displayName: displayName.trim()
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("workspace.setupAdmin.title")}</DialogTitle>
          <DialogDescription>{t("workspace.setupAdmin.description", { serverUrl })}</DialogDescription>
        </DialogHeader>

        <form className="space-y-3" onSubmit={(event) => void handleSubmit(event)}>
          <label className="grid gap-1 text-small text-muted-foreground">
            {t("workspace.setupAdmin.bootstrapToken")}
            <Input value={bootstrapToken} onChange={(event) => setBootstrapToken(event.target.value)} required />
          </label>

          <label className="grid gap-1 text-small text-muted-foreground">
            {t("workspace.setupAdmin.displayName")}
            <Input value={displayName} onChange={(event) => setDisplayName(event.target.value)} required />
          </label>

          <label className="grid gap-1 text-small text-muted-foreground">
            {t("workspace.setupAdmin.email")}
            <Input type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
          </label>

          <label className="grid gap-1 text-small text-muted-foreground">
            {t("workspace.setupAdmin.password")}
            <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
          </label>

          {errorMessage ? <p className="text-small text-destructive">{errorMessage}</p> : null}

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("workspace.cancel")}
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? t("workspace.loading") : t("workspace.setupAdmin.submit")}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
