import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { afterEach, describe, expect, it } from "vitest";

import BaseModal from "@/shared/ui/BaseModal.vue";

afterEach(() => {
  document.body.innerHTML = "";
});

describe("base modal", () => {
  it("renders dialog semantics and focuses first focusable element", async () => {
    const wrapper = mount(BaseModal, {
      attachTo: document.body,
      props: {
        open: true,
        ariaLabel: "测试弹窗"
      },
      slots: {
        title: "<h3>测试弹窗</h3>",
        default: "<input id='modal-first-input' /><button id='modal-button'>操作</button>",
        footer: "<button>关闭</button>"
      }
    });

    await nextTick();

    const dialog = wrapper.get("[role='dialog']");
    expect(dialog.attributes("aria-modal")).toBe("true");
    expect(dialog.attributes("aria-label")).toBe("测试弹窗");
    expect(document.activeElement).toBe(wrapper.get("#modal-first-input").element);

    wrapper.unmount();
  });

  it("traps focus with tab and shift+tab", async () => {
    const wrapper = mount(BaseModal, {
      attachTo: document.body,
      props: {
        open: true
      },
      slots: {
        title: "<h3>焦点陷阱</h3>",
        default: "<button id='first-focus'>第一个</button><button id='last-focus'>最后一个</button>"
      }
    });

    await nextTick();

    const dialog = wrapper.get("[role='dialog']");
    const firstButton = wrapper.get("#first-focus").element as HTMLButtonElement;
    const lastButton = wrapper.get("#last-focus").element as HTMLButtonElement;

    lastButton.focus();
    await dialog.trigger("keydown", { key: "Tab" });
    expect(document.activeElement).toBe(firstButton);

    firstButton.focus();
    await dialog.trigger("keydown", { key: "Tab", shiftKey: true });
    expect(document.activeElement).toBe(lastButton);

    wrapper.unmount();
  });

  it("emits close on escape and restores previous focus when closed", async () => {
    const trigger = document.createElement("button");
    trigger.textContent = "open";
    document.body.appendChild(trigger);
    trigger.focus();

    const wrapper = mount(BaseModal, {
      attachTo: document.body,
      props: {
        open: true
      },
      slots: {
        title: "<h3>关闭行为</h3>",
        default: "<button>inside</button>",
        footer: "<button>关闭</button>"
      }
    });

    await nextTick();

    await wrapper.get("[role='dialog']").trigger("keydown", { key: "Escape" });
    expect(wrapper.emitted("close")?.length ?? 0).toBe(1);

    await wrapper.setProps({ open: false });
    await nextTick();

    expect(document.activeElement).toBe(trigger);

    wrapper.unmount();
  });
});
