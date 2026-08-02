export default {
  extends: [
    'stylelint-config-standard',
    'stylelint-config-standard-scss',
    'stylelint-config-recommended-vue/scss',
  ],
  overrides: [
    {
      files: ['**/*.vue'],
      customSyntax: 'postcss-html',
      rules: {
        // stylelint 17.14.1 起 no-invalid-position-declaration 会对模板内联
        // style="…" 属性误报（postcss-html 解析的裸声明本就没有规则上下文）。
        // 仅对 .vue 关闭；.css/.scss 仍保留该规则。
        'no-invalid-position-declaration': null,
      },
    },
    { files: ['**/*.scss'], customSyntax: 'postcss-scss' },
  ],
  rules: {
    'no-descending-specificity': null,
    'no-empty-source': null,
    // BEM convention (__element, --modifier) + allow uppercase for third-party class targeting
    'selector-class-pattern': null,
    // Preserve author-intended blank lines (used as section dividers in theme token files)
    'declaration-empty-line-before': null,
    'custom-property-empty-line-before': null,
    // Keep `currentColor` camelCase; both spellings are spec-valid
    'value-keyword-case': null,
    // Vue 2 legacy deep selectors (Vue 3 `:deep`/`:slotted`/`:global` already handled by recommended-vue/scss)
    'selector-pseudo-element-no-unknown': [
      true,
      { ignorePseudoElements: ['v-deep', 'v-slotted', 'v-global'] },
    ],
    'at-rule-no-unknown': [
      true,
      {
        ignoreAtRules: [
          // UnoCSS / Tailwind directives
          'apply',
          'screen',
          'variants',
          'tailwind',
          // SCSS directives (also needed here because the base rule still runs over .scss sources)
          'use',
          'forward',
          'mixin',
          'include',
          'if',
          'else',
          'each',
          'for',
          'function',
          'return',
        ],
      },
    ],
  },
  ignoreFiles: ['dist/**', 'node_modules/**', 'public/**', 'src/**/*.d.ts'],
}
