<template>
  <component
    :is="resolvedIcon"
    :size="size"
    :stroke-width="strokeWidth"
    class="app-icon"
    aria-hidden="true"
  />
</template>

<script setup lang="ts">
import {
  ArrowUp,
  Bot,
  BriefcaseBusiness,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  Circle,
  CircleUserRound,
  Clock3,
  CornerDownRight,
  Cpu,
  FileDown,
  FileText,
  Folder,
  FolderOpen,
  House,
  Info,
  Pencil,
  PlugZap,
  Plus,
  ScrollText,
  SendHorizontal,
  Settings,
  ShieldCheck,
  Square,
  Trash2,
  User,
  UserRound,
  Users,
  Wrench
} from "lucide-vue-next";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    name: string;
    size?: number;
    strokeWidth?: number;
  }>(),
  {
    size: 14,
    strokeWidth: 2
  }
);

const iconMap = {
  "arrow-up": ArrowUp,
  bot: Bot,
  "briefcase-business": BriefcaseBusiness,
  "chevron-down": ChevronDown,
  "chevron-left": ChevronLeft,
  "chevron-right": ChevronRight,
  "chevron-up": ChevronUp,
  circle: Circle,
  "circle-user-round": CircleUserRound,
  "clock-3": Clock3,
  "corner-down-right": CornerDownRight,
  cpu: Cpu,
  "file-down": FileDown,
  "file-text": FileText,
  folder: Folder,
  "folder-open": FolderOpen,
  house: House,
  info: Info,
  pencil: Pencil,
  "plug-zap": PlugZap,
  plus: Plus,
  "scroll-text": ScrollText,
  "send-horizontal": SendHorizontal,
  settings: Settings,
  "shield-check": ShieldCheck,
  square: Square,
  "trash-2": Trash2,
  user: User,
  "user-round": UserRound,
  users: Users,
  wrench: Wrench
} as const;

const resolvedIcon = computed(() => iconMap[props.name as keyof typeof iconMap] ?? Square);
</script>

<style scoped>
.app-icon {
  display: inline-flex;
  flex-shrink: 0;
}
</style>
