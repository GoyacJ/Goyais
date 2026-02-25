import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { afterEach, describe, expect, it } from "vitest";

import WorkspaceCreateModal from "@/shared/ui/sidebar/WorkspaceCreateModal.vue";

afterEach(() => {
  document.body.innerHTML = "";
});

describe("workspace create modal", () => {
  it("supports keyboard close via Escape and restores trigger focus", async () => {
    const trigger = document.createElement("button");
    trigger.textContent = "open";
    document.body.appendChild(trigger);
    trigger.focus();

    const wrapper = mount(WorkspaceCreateModal, {
      attachTo: document.body,
      props: {
        open: true
      }
    });

    await nextTick();

    expect(wrapper.find("[data-testid='workspace-create-modal']").exists()).toBe(true);

    await wrapper.get("[role='dialog']").trigger("keydown", { key: "Escape" });
    expect(wrapper.emitted("close")?.length ?? 0).toBe(1);

    await wrapper.setProps({ open: false });
    await nextTick();

    expect(document.activeElement).toBe(trigger);

    wrapper.unmount();
  });
});
