<template>
  <section
    class="grid gap-[var(--global-space-12)] border border-[var(--component-card-border)] rounded-[var(--component-card-radius)] bg-[var(--component-card-bg)] p-[var(--component-card-padding)]"
  >
    <header v-if="$slots.header" class="text-[var(--component-card-header-fg)]">
      <slot name="header" />
    </header>
    <div class="text-[var(--component-card-body-fg)]">
      <slot />
    </div>
    <footer
      v-if="$slots.footer"
      class="rounded-[var(--global-radius-8)] bg-[var(--component-card-footer-bg)] px-[var(--global-space-12)] py-[var(--global-space-8)]"
    >
      <slot name="footer" />
    </footer>
  </section>
</template>
