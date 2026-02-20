import { useTranslation } from "react-i18next";

import { ErrorState } from "@/components/domain/feedback/ErrorState";

export function NoPermissionPage() {
  const { t } = useTranslation();

  return (
    <ErrorState
      title={t("workspace.noPermission.title")}
      message={t("workspace.noPermission.message")}
    />
  );
}
