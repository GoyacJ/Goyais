import { LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

interface LoadingStateProps {
  label?: string;
}

export function LoadingState({ label }: LoadingStateProps) {
  const { t } = useTranslation();

  return (
    <div className="flex h-full min-h-[8rem] items-center justify-center gap-2 rounded-control border border-border-subtle bg-background/40 p-3 text-small text-muted-foreground">
      <LoaderCircle className="h-4 w-4 animate-spin" />
      <span>{label ?? t("feedback.loading")}</span>
    </div>
  );
}
