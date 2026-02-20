import { type ReactNode } from "react";

import { ToastStateProvider } from "@/components/ui/toast";
import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";

interface AppProvidersProps {
  children: ReactNode;
}

export function AppProviders({ children }: AppProvidersProps) {
  return (
    <TooltipProvider delayDuration={200}>
      <ToastStateProvider>
        {children}
        <Toaster />
      </ToastStateProvider>
    </TooltipProvider>
  );
}
