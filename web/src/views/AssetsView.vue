<template>
  <section class="ui-page">
    <PageHeader :title="t('page.assets.title')" :subtitle="t('page.assets.subtitle')" />

    <SectionCard title="Assets Table" subtitle="Compact console baseline">
      <Table :columns="columns" :rows="rows" />
      <div class="mt-3">
        <Pagination :page="page" :page-size="10" :total="20" @update:page="page = $event" />
      </div>
    </SectionCard>

    <EmptyState :description="t('common.empty')" />
  </section>
</template>

<script setup lang="ts">
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Table from '@/components/ui/Table.vue'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const page = ref(1)

const columns = [
  { key: 'assetId', label: 'assetId', mono: true },
  { key: 'assetType', label: 'assetType' },
  { key: 'visibility', label: 'visibility' },
]

const rows = [
  { assetId: 'ast_01hz8f2', assetType: 'video', visibility: 'WORKSPACE' },
  { assetId: 'ast_01hz8fx', assetType: 'image', visibility: 'PRIVATE' },
]
</script>
