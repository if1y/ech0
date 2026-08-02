// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { fetchQueryEchos, fetchGetTags, fetchGetEchoById, fetchCreateTag } from '@/service/api'

export const useEchoStore = defineStore('echoStore', () => {
  const normalizeEchoId = (echo: App.Api.Ech0.Echo): string => String(echo?.id ?? '').trim()

  // ─────────────────────────────────────────────
  //  统一查询状态（普通时间线与标签过滤共享同一套状态）
  // ─────────────────────────────────────────────

  const echoList = ref<App.Api.Ech0.Echo[]>([]) // 当前页展示的 Echo 列表
  const echoIndexMap = ref(new Map<string, number>()) // id → echoList 下标，用于快速定位
  const isLoading = ref<boolean>(true)
  const total = ref<number>(0) // 当前查询条件下的总条数
  const pageSize = ref<number>(7) // 每页数量
  const currentPage = ref<number>(1) // 当前页码（从 1 开始）
  const searchValue = ref<string>('')
  const searchingMode = computed(() => searchValue.value.length > 0)
  const totalPages = computed(() =>
    total.value <= 0 ? 1 : Math.max(1, Math.ceil(total.value / pageSize.value)),
  )
  const echoToUpdate = ref<App.Api.Ech0.EchoToUpdate | null>(null)

  const tagList = ref<App.Api.Ech0.Tag[]>([])
  const tagOptions = computed<string[]>(() => tagList.value.map((tag) => tag.name))

  // ── 标签过滤模式 ──
  const isFilteringMode = ref<boolean>(false)
  const filteredTag = ref<App.Api.Ech0.Tag | null>(null)

  // ── 日期范围过滤（由高级搜索面板驱动；单位：Unix 秒） ──
  const dateFrom = ref<number | null>(null)
  const dateTo = ref<number | null>(null)
  const isDateRangeActive = computed(() => dateFrom.value !== null || dateTo.value !== null)

  // ── 多标签过滤（由高级搜索面板驱动）──
  const selectedTagIds = ref<string[]>([])
  const isTagSelectionActive = computed(() => selectedTagIds.value.length > 0)

  // ── 可见性过滤（由高级搜索面板驱动；仅 admin 有效，服务端对无权限请求静默忽略）──
  const visibilityFilter = ref<App.Api.Ech0.EchoVisibilityFilter>('all')
  const isVisibilityFilterActive = computed(() => visibilityFilter.value !== 'all')

  // ─────────────────────────────────────────────
  //  watchers
  // ─────────────────────────────────────────────

  watch(searchingMode, (newValue, oldValue) => {
    if (newValue === false && oldValue === true) {
      refreshEchos()
    }
  })

  // ─────────────────────────────────────────────
  //  核心查询 actions
  // ─────────────────────────────────────────────

  function buildQueryParams(): App.Api.Ech0.EchoQueryParams {
    const params: App.Api.Ech0.EchoQueryParams = {
      page: currentPage.value,
      pageSize: pageSize.value,
      search: searchValue.value || undefined,
    }
    const tagIds = new Set<string>()
    if (isFilteringMode.value && filteredTag.value) {
      tagIds.add(filteredTag.value.id)
    }
    selectedTagIds.value.forEach((id) => tagIds.add(id))
    if (tagIds.size > 0) {
      params.tagIds = Array.from(tagIds)
    }
    if (dateFrom.value !== null) {
      params.dateFrom = dateFrom.value
    }
    if (dateTo.value !== null) {
      params.dateTo = dateTo.value
    }
    if (visibilityFilter.value !== 'all') {
      params.private = visibilityFilter.value === 'private'
    }
    return params
  }

  const resetDateRange = () => {
    dateFrom.value = null
    dateTo.value = null
  }

  const resetSelectedTags = () => {
    selectedTagIds.value = []
  }

  const resetVisibilityFilter = () => {
    visibilityFilter.value = 'all'
  }

  const removeSelectedTag = (tagId: string) => {
    selectedTagIds.value = selectedTagIds.value.filter((id) => id !== tagId)
    if (filteredTag.value?.id === tagId && isFilteringMode.value) {
      isFilteringMode.value = false
      filteredTag.value = null
    }
  }

  /**
   * 拉取当前页数据，整页替换 echoList。
   * 同一查询条件并发调用复用同一个 Promise，避免重复请求。
   */
  let pendingFetch: Promise<void> | null = null
  async function fetchCurrentPage() {
    if (pendingFetch) return pendingFetch

    isLoading.value = true
    pendingFetch = fetchQueryEchos(buildQueryParams())
      .then((res) => {
        if (res.code === 1) {
          total.value = res.data.total
          const items = (res.data.items ?? []).map((item) => ({
            ...item,
            id: normalizeEchoId(item),
          }))
          echoList.value = items
          const nextIndex = new Map<string, number>()
          items.forEach((item, idx) => {
            if (item.id) nextIndex.set(item.id, idx)
          })
          echoIndexMap.value = nextIndex
        }
      })
      .finally(() => {
        isLoading.value = false
        pendingFetch = null
      })

    return pendingFetch
  }

  /**
   * 跳转到指定页（页码会被夹到 [1, totalPages]）。
   * 若与当前页相同则跳过请求。
   */
  async function goToPage(page: number) {
    const target = Math.max(1, Math.floor(page) || 1)
    if (target === currentPage.value && echoList.value.length > 0) return
    currentPage.value = target
    await fetchCurrentPage()
  }

  /** 重置分页指针并重新拉取第一页。 */
  const refreshEchos = () => {
    currentPage.value = 1
    total.value = 0
    echoList.value = []
    echoIndexMap.value = new Map()
    return fetchCurrentPage()
  }

  /** 清空列表但不发起请求（用于登出 / 卸载等场景）。 */
  const clearEchos = () => {
    currentPage.value = 1
    total.value = 0
    echoList.value = []
    echoIndexMap.value = new Map()
  }

  /** 仅重置分页指针与列表，不触发加载（搜索场景由调用方控制加载时机）。 */
  const refreshForSearch = () => {
    currentPage.value = 1
    total.value = 0
    echoList.value = []
    echoIndexMap.value = new Map()
  }

  const updateEcho = (echo: App.Api.Ech0.Echo) => {
    const idx = echoIndexMap.value.get(echo.id)
    if (idx !== undefined) {
      echoList.value[idx] = echo
    }
  }

  const updateLikeCount = (echoId: string, delta: number = 1) => {
    const idx = echoIndexMap.value.get(echoId)
    if (idx !== undefined) {
      const targetEcho = echoList.value[idx]
      if (targetEcho) {
        targetEcho.fav_count = (targetEcho.fav_count || 0) + delta
        echoList.value[idx] = { ...targetEcho }
      }
    }
  }

  // 单条 Echo 的飞行中请求池：让 echo API 与 EchoView chunk 下载并行，
  // 避免分享链接首次访问的串行等待。命中当前页缓存时立即返回。
  const pendingEchoMap = new Map<string, Promise<App.Api.Ech0.Echo | null>>()

  const prefetchEcho = (echoId: string): Promise<App.Api.Ech0.Echo | null> => {
    const id = String(echoId ?? '').trim()
    if (!id) return Promise.resolve(null)

    const idx = echoIndexMap.value.get(id)
    if (idx !== undefined && echoList.value[idx]) {
      return Promise.resolve(echoList.value[idx]!)
    }

    const existing = pendingEchoMap.get(id)
    if (existing) return existing

    const promise: Promise<App.Api.Ech0.Echo | null> = fetchGetEchoById(id)
      .then((res) => {
        if (res.code === 1 && res.data) {
          return res.data
        }
        return null
      })
      .catch(() => null)
      .finally(() => {
        pendingEchoMap.delete(id)
      }) as Promise<App.Api.Ech0.Echo | null>

    pendingEchoMap.set(id, promise)
    return promise
  }

  const getTags = async () => {
    const res = await fetchGetTags()
    if (res.code === 1) {
      tagList.value.splice(0, tagList.value.length, ...res.data)
    }
  }

  const createTag = async (name: string) => {
    const res = await fetchCreateTag(name)
    if (res.code === 1) {
      await getTags()
      return res.data
    }
    return null
  }

  let tagsLoadPromise: Promise<void> | null = null
  const ensureTagsLoaded = (): Promise<void> => {
    if (tagList.value.length > 0) return Promise.resolve()
    if (tagsLoadPromise) return tagsLoadPromise
    tagsLoadPromise = getTags().finally(() => {
      tagsLoadPromise = null
    })
    return tagsLoadPromise
  }

  return {
    // 状态
    echoList,
    echoIndexMap,
    isLoading,
    total,
    pageSize,
    currentPage,
    totalPages,
    searchValue,
    searchingMode,
    echoToUpdate,
    tagList,
    tagOptions,

    // 标签过滤
    isFilteringMode,
    filteredTag,

    // 日期范围过滤
    dateFrom,
    dateTo,
    isDateRangeActive,

    // 多标签过滤
    selectedTagIds,
    isTagSelectionActive,

    // 可见性过滤
    visibilityFilter,
    isVisibilityFilterActive,

    // actions
    fetchCurrentPage,
    goToPage,
    refreshEchos,
    clearEchos,
    refreshForSearch,
    resetDateRange,
    resetSelectedTags,
    resetVisibilityFilter,
    removeSelectedTag,
    updateEcho,
    updateLikeCount,
    prefetchEcho,
    getTags,
    ensureTagsLoaded,
    createTag,
  }
})
