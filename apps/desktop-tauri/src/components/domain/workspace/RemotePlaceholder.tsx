import { useTranslation } from "react-i18next";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

interface RemotePlaceholderProps {
  section: "run" | "projects" | "models" | "replay";
}

export function RemotePlaceholder({ section }: RemotePlaceholderProps) {
  const { t } = useTranslation();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t(`workspace.remotePlaceholder.${section}.title`)}</CardTitle>
        <CardDescription>{t("workspace.remotePlaceholder.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-small text-muted-foreground">{t("workspace.remotePlaceholder.nextStep")}</p>
      </CardContent>
    </Card>
  );
}
