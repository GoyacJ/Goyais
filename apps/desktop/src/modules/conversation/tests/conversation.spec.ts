import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import ConversationView from "@/modules/conversation/views/ConversationView.vue";

describe("conversation view", () => {
  it("renders required placeholders", () => {
    const wrapper = mount(ConversationView);

    expect(wrapper.find('[data-testid="conversation-sidebar"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="conversation-events"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="conversation-composer"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="conversation-stop-button"]').exists()).toBe(true);
  });
});
