import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import BaseResourceCard from "@/shared/ui/BaseResourceCard.vue";

describe("base resource card", () => {
  it("renders title status and actions with slot groups", () => {
    const wrapper = mount(BaseResourceCard, {
      slots: {
        title: "<span>GitHub MCP</span>",
        status: "<span>启用</span>",
        default: "<p>传输：http_sse</p>",
        actionsPrimary: "<button>连接</button><button>编辑</button>",
        actionsSecondary: "<button>停用</button><button>删除</button>"
      }
    });

    expect(wrapper.text()).toContain("GitHub MCP");
    expect(wrapper.text()).toContain("启用");
    expect(wrapper.text()).toContain("传输：http_sse");

    const primary = wrapper.find(".actions-primary");
    const secondary = wrapper.find(".actions-secondary");
    expect(primary.exists()).toBe(true);
    expect(secondary.exists()).toBe(true);
    expect(primary.text()).toContain("连接");
    expect(primary.text()).toContain("编辑");
    expect(secondary.text()).toContain("停用");
    expect(secondary.text()).toContain("删除");
  });

  it("renders details block with custom label and keeps collapsed by default", async () => {
    const wrapper = mount(BaseResourceCard, {
      props: {
        detailsLabel: "连接详情"
      },
      slots: {
        title: "<span>Workspace MCP</span>",
        default: "<p>最近探测：未探测</p>",
        details: "<p>错误码：http_401</p>"
      }
    });

    const details = wrapper.find("details");
    expect(details.exists()).toBe(true);
    expect(wrapper.find("summary").text()).toBe("连接详情");
    expect((details.element as HTMLDetailsElement).open).toBe(false);

    (details.element as HTMLDetailsElement).open = true;
    await wrapper.vm.$nextTick();

    expect((details.element as HTMLDetailsElement).open).toBe(true);
    expect(wrapper.text()).toContain("错误码：http_401");
  });
});
