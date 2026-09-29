<script setup lang="ts">
defineProps<{
  title: string
  subtitle: string
  code: string
}>()
</script>

<template>
  <details class="definition-accordion">
    <summary class="definition-accordion__summary">
      <span class="definition-accordion__copy">
        <strong>{{ title }}</strong>
        <small>{{ subtitle }}</small>
      </span>
      <span class="definition-accordion__chevron" aria-hidden="true">›</span>
    </summary>
    <div class="definition-accordion__content">
      <pre><code>{{ code }}</code></pre>
    </div>
  </details>
</template>

<style scoped>
.definition-accordion {
  margin: 0 auto 1rem;
  max-width: 72rem;
  border: 1px solid rgb(31 41 51 / 13%);
  border-radius: .625rem;
  background: rgb(255 254 251 / 82%);
  color: #1f2933;
}

.definition-accordion__summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: .75rem;
  padding: .75rem 1rem;
  cursor: pointer;
  list-style: none;
}

.definition-accordion__summary::-webkit-details-marker { display: none; }
.definition-accordion__summary:hover { background: rgb(31 111 95 / 6%); }

.definition-accordion__copy { display: grid; gap: .125rem; }
.definition-accordion__copy strong { font-size: .9375rem; }
.definition-accordion__copy small { color: #52606d; font-size: .8125rem; }

.definition-accordion__chevron {
  flex: 0 0 auto;
  color: #1f6f5f;
  font-size: 1.65rem;
  line-height: 1;
  transform: rotate(0deg);
  transition: transform 120ms ease-out;
}

.definition-accordion[open] .definition-accordion__chevron { transform: rotate(90deg); }

.definition-accordion__content {
  border-top: 1px solid rgb(31 41 51 / 9%);
  padding: .75rem 1rem 1rem;
}

.definition-accordion pre {
  max-height: 24rem;
  margin: 0;
  overflow: auto;
  border-radius: .375rem;
  background: #1f2933;
  color: #edf2f7;
  font: .8125rem/1.55 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  padding: .875rem 1rem;
  tab-size: 2;
  white-space: pre;
}

@media (max-width: 44rem) {
  .definition-accordion { margin-bottom: .75rem; }
  .definition-accordion__summary { padding: .7rem .8rem; }
  .definition-accordion__content { padding: .65rem .8rem .8rem; }
  .definition-accordion pre { max-height: 20rem; padding: .75rem; font-size: .75rem; }
}

:global(body:has([data-clear-crud-theme="dark"])) .definition-accordion {
  border-color: rgb(237 242 247 / 16%);
  background: rgb(30 37 44 / 92%);
  color: #edf2f7;
}

:global(body:has([data-clear-crud-theme="dark"])) .definition-accordion__copy small { color: #b7c2cd; }
:global(body:has([data-clear-crud-theme="dark"])) .definition-accordion__content { border-top-color: rgb(237 242 247 / 11%); }
</style>
