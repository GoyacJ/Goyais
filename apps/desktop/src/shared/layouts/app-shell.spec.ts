import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";

import AppShell from "@/shared/layouts/AppShell.vue";
import { authStore, resetAuthStore } from "@/shared/stores/authStore";

describe("app shell", () => {
  beforeEach(() => {
    resetAuthStore();
  });

  it("hides admin link when admin capability is false", () => {
    authStore.capabilities = {
      admin_console: false,
      resource_write: false,
      execution_control: false
    };

    const wrapper = mountShell();
    expect(wrapper.text()).not.toContain("Admin");
  });

  it("shows admin link when admin capability is true", () => {
    authStore.capabilities = {
      admin_console: true,
      resource_write: true,
      execution_control: true
    };

    const wrapper = mountShell();
    expect(wrapper.text()).toContain("Admin");
  });
});

function mountShell() {
  return mount(AppShell, {
    slots: {
      default: "<div>content</div>"
    },
    global: {
      stubs: {
        RouterLink: {
          template: "<a><slot /></a>"
        }
      }
    }
  });
}
