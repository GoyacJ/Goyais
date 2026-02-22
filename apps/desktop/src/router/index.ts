import { createRouter, createWebHistory } from "vue-router";

import AdminView from "@/modules/admin/views/AdminView.vue";
import ConversationView from "@/modules/conversation/views/ConversationView.vue";
import ProjectView from "@/modules/project/views/ProjectView.vue";
import ResourceView from "@/modules/resource/views/ResourceView.vue";
import WorkspaceView from "@/modules/workspace/views/WorkspaceView.vue";

export const routes = [
  { path: "/", redirect: "/conversation" },
  { path: "/workspace", name: "workspace", component: WorkspaceView },
  { path: "/project", name: "project", component: ProjectView },
  { path: "/conversation", name: "conversation", component: ConversationView },
  { path: "/resource", name: "resource", component: ResourceView },
  { path: "/admin", name: "admin", component: AdminView }
];

export const router = createRouter({
  history: createWebHistory(),
  routes
});
