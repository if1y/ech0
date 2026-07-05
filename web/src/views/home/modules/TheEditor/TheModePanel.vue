<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Copyright (C) 2025-2026 lin-snow -->
<template>
  <div
    class="editor-mode-panel p-3 my-3 border border-dashed border-[var(--dash-line-color)] rounded-xs"
  >
    <!-- 扩展附加内容 -->
    <div class="mb-1">
      <h2 class="text-[var(--color-text-muted)] font-bold mb-1">{{ t('editor.extraContent') }}</h2>
      <div class="flex flex-row items-center gap-2">
        <!-- 添加音乐 -->
        <BaseButton
          :icon="Music"
          class="w-7 h-7 rounded-xs"
          :tooltip="t('editor.addMusic')"
          @click="handleAddExtension(ExtensionType.MUSIC)"
        />
        <!-- 添加视频 -->
        <BaseButton
          :icon="Video"
          class="w-7 h-7 rounded-xs"
          :tooltip="t('editor.addVideo')"
          @click="handleAddExtension(ExtensionType.VIDEO)"
        />
        <!-- 添加Github项目 -->
        <BaseButton
          :icon="Githubproj"
          class="w-7 h-7 rounded-xs"
          :tooltip="t('editor.addGithubProject')"
          @click="handleAddExtension(ExtensionType.GITHUBPROJ)"
        />
        <!-- 添加网站链接 -->
        <BaseButton
          :icon="Weblink"
          class="w-7 h-7 rounded-xs"
          :tooltip="t('editor.addWebsiteLink')"
          @click="handleAddExtension(ExtensionType.WEBSITE)"
        />
        <!-- 添加位置 -->
        <BaseButton
          :icon="MapPin"
          class="w-7 h-7 rounded-xs"
          :tooltip="t('editor.addLocation')"
          @click="handleAddExtension(ExtensionType.LOCATION)"
        />
        <!-- 添加 X 推文 -->
        <BaseButton
          :icon="XBrand"
          class="w-7 h-7 rounded-xs"
          :tooltip="t('editor.addTweet')"
          @click="handleAddExtension(ExtensionType.TWEET)"
        />
      </div>
    </div>
    <!-- 自定义发表时间 -->
    <div>
      <h2 class="text-[var(--color-text-muted)] font-bold mb-1">{{ t('editor.customCreatedAt') }}</h2>
      <p class="text-xs text-[var(--color-text-muted)] mb-1">{{ t('editor.customCreatedAtFormat') }}</p>
      <input
        v-model="editorStore.customCreatedAt"
        type="text"
        :placeholder="t('editor.customCreatedAtPlaceholder')"
        :class="[
          'w-full px-2 py-1 text-sm bg-[var(--md-editor-mini-btn-bg)] border rounded-xs outline-none transition-colors',
          editorStore.isCustomCreatedAtValid
            ? 'border-[var(--md-editor-mini-border)] focus:border-[var(--color-primary)]'
            : 'border-red-500 focus:border-red-500',
        ]"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import Weblink from '@/components/icons/weblink.vue'
import Music from '@/components/icons/music.vue'
import Video from '@/components/icons/video.vue'
import Githubproj from '@/components/icons/githubproj.vue'
import MapPin from '@/components/icons/mappin.vue'
import XBrand from '@/components/icons/tweet.vue'
import BaseButton from '@/components/common/BaseButton.vue'

import { Mode, ExtensionType } from '@/enums/enums'
import { useEditorStore } from '@/stores'
import { useI18n } from 'vue-i18n'

const editorStore = useEditorStore()
const { t } = useI18n()

const handleAddExtension = (extensiontype: ExtensionType) => {
  editorStore.currentMode = Mode.EXTEN
  editorStore.currentExtensionType = extensiontype
  editorStore.extensionToAdd.extension_type = extensiontype
}
</script>

<style scoped>
.editor-mode-panel {
  border-color: var(--md-editor-mini-border);

  --btn-bg-color: var(--md-editor-mini-btn-bg);
  --btn-ring-color: var(--md-editor-actions-ring-color);
  --btn-hover-bg-color: var(--md-editor-actions-hover-bg);
  --btn-hover-border-color: var(--md-editor-actions-hover-border);
}
</style>
