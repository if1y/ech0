<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Copyright (C) 2025-2026 lin-snow -->
<template>
  <BaseTooltip :content="effectiveTooltip">
    <button
      :class="[
        'relative cursor-pointer p-1.5 rounded-[var(--btn-radius)] ring-inset ring-1 ring-[var(--btn-ring-color)] text-[var(--btn-text-color)] outline-none shadow-[var(--btn-shadow)] transition-colors duration-200',
        hasBg ? '' : 'bg-[var(--btn-bg-color)]',
        isDisabled
          ? 'cursor-not-allowed opacity-70'
          : 'hover:bg-[var(--btn-hover-bg-color)] hover:ring-[var(--btn-hover-border-color)] focus-visible:ring-2 focus-visible:ring-[var(--btn-focus-ring-color)]',
        props.class,
      ]"
      :type="type"
      :disabled="isDisabled"
      :aria-label="resolvedAriaLabel"
      @click="onClick"
    >
      <!-- 内容（图标 / 文字）始终保留在流中：loading 时仅置为 invisible，撑住按钮原尺寸，避免布局抖动 -->
      <span v-if="icon" class="flex items-center justify-center" :class="{ invisible: loading }">
        <component :is="icon" class="w-full h-full" />
      </span>
      <span v-if="$slots.default" :class="{ invisible: loading }"><slot /></span>
      <!-- 菊花绝对定位覆盖在按钮正中，不占流，故不影响宽高 -->
      <span v-if="loading" class="absolute inset-0 flex items-center justify-center">
        <TheLoadingIndicator size="sm" :center="false" />
      </span>
    </button>
  </BaseTooltip>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { computed } from 'vue'
import BaseTooltip from './BaseTooltip.vue'
import TheLoadingIndicator from './TheLoadingIndicator.vue'

const props = defineProps<{
  icon?: Component
  disabled?: boolean
  loading?: boolean
  class?: string // 接收父组件传递的 class
  tooltip?: string
  title?: string
  ariaLabel?: string
  type?: 'button' | 'submit' | 'reset'
}>()

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()

// const customClass = props.class
const hasBg = computed(() => props.class?.includes('bg-') || props.class?.includes('!bg-'))

// loading 等同于不可点击：合并进禁用态，避免重复传 disabled
const isDisabled = computed(() => props.disabled || props.loading)

const resolvedTooltip = computed(() => props.tooltip ?? props.title ?? '')
const effectiveTooltip = computed(() => resolvedTooltip.value)
const resolvedAriaLabel = computed(() => props.ariaLabel ?? (effectiveTooltip.value || undefined))
const type = computed(() => props.type ?? 'button')

function onClick(event: MouseEvent) {
  if (!isDisabled.value) emit('click', event)
}
</script>

<style scoped></style>
