<template>
  <section class="interactive-model" aria-label="Интерактивная модель срезов Go">
    <label class="model-select-label">
      Сценарий
      <select :value="state.scenarioId" aria-label="Сценарий модели срезов" @change="selectScenario">
        <option v-for="scenario in scenarios" :key="scenario.id" :value="scenario.id">{{ scenario.label }}</option>
      </select>
    </label>

    <div class="model-controls" role="group" aria-label="Управление моделью срезов">
      <button type="button" :disabled="state.finished" @click="step">Следующий шаг</button>
      <button type="button" @click="reset">Сбросить сценарий</button>
    </div>

    <p class="model-summary">{{ currentScenario.summary }}</p>
    <p class="model-version">{{ state.goVersion }} · шаг {{ state.stepIndex + 1 }}</p>
    <p class="model-live-status" role="status" aria-live="polite" aria-atomic="true">{{ statusAnnouncement }}</p>
    <pre class="model-code"><code>{{ state.code }}</code></pre>

    <svg
      class="model-svg"
      :viewBox="`0 0 360 ${diagramHeight}`"
      role="img"
      :aria-label="diagramLabel"
    >
      <g v-for="(array, arrayIndex) in state.view.backingArrays" :key="array.id">
        <text class="svg-label" x="12" :y="34 + arrayIndex * 112">Массив {{ array.id }}</text>
        <g
          v-for="(cell, cellIndex) in array.cells"
          :key="`${array.id}-${cellIndex}`"
          :transform="'translate(' + (84 + cellIndex * 48) + ' ' + (14 + arrayIndex * 112) + ')'"
        >
          <rect class="svg-cell" width="42" height="36" rx="5" />
          <text class="svg-value" x="21" y="23" text-anchor="middle">{{ cell === null ? '·' : cell }}</text>
          <text class="svg-index" x="21" y="54" text-anchor="middle">[{{ cellIndex }}]</text>
        </g>
        <text class="svg-caption" x="84" :y="96 + arrayIndex * 112">
          capacity: {{ array.capacity === null ? '≥ ' + array.cells.length : array.capacity }}
        </text>
      </g>
      <g v-for="(slice, index) in state.view.slices" :key="slice.name">
        <path
          class="svg-reference"
          :d="referencePath(slice, index)"
        />
        <text class="svg-label" x="12" :y="referenceY(index)">
          {{ slice.name }}
        </text>
      </g>
    </svg>

    <div class="model-text-state">
      <p>{{ state.explanation }}</p>
      <ul>
        <li v-for="array in state.view.backingArrays" :key="array.id">
          Массив {{ array.id }}: [{{ array.cells.map((cell) => cell === null ? 'пусто' : cell).join(', ') }}],
          capacity {{ array.capacity === null ? 'не меньше длины (' + array.cells.length + '); точное значение зависит от реализации' : array.capacity }}.
        </li>
        <li v-for="slice in state.view.slices" :key="slice.name">
          {{ slice.name }} ссылается на {{ slice.backingArray }}, len={{ slice.length }},
          cap={{ slice.capacity === null ? 'не меньше len (' + slice.length + '); точное значение зависит от реализации' : slice.capacity }}.
        </li>
      </ul>
      <p class="model-assumptions">{{ state.assumptions }}</p>
    </div>
  </section>
</template>

<script>
import * as model from '../models/slices.mjs';

export default {
  name: 'SlicesModel',
  data() {
    return {
      scenarios: model.scenarios,
      state: model.createState(model.scenarios[0].id),
      announcementAction: 'Текущий сценарий.',
    };
  },
  computed: {
    statusAnnouncement() {
      return this.announcementAction + ' Шаг ' + (this.state.stepIndex + 1) + ': ' + this.state.explanation;
    },
    currentScenario() {
      return this.scenarios.find(({ id }) => id === this.state.scenarioId);
    },
    diagramHeight() {
      return 30 + this.state.view.backingArrays.length * 112 + this.state.view.slices.length * 26;
    },
    diagramLabel() {
      const arrays = this.state.view.backingArrays.length;
      const refs = this.state.view.slices.map(({ name, backingArray }) => `${name} указывает на массив ${backingArray}`).join('; ');
      return `${arrays} backing array; ${refs}`;
    },
  },
  methods: {
    selectScenario(event) {
      this.state = model.createState(event.target.value);
      this.announcementAction = 'Выбран сценарий ' + this.currentScenario.label + '.';
    },
    step() {
      this.state = model.next(this.state);
      this.announcementAction = 'Показан следующий шаг.';
    },
    reset() {
      this.state = model.reset(this.state);
      this.announcementAction = 'Сценарий сброшен.';
    },
    referenceY(index) {
      return 18 + this.state.view.backingArrays.length * 112 + index * 26;
    },
    referencePath(slice, index) {
      const arrayIndex = this.state.view.backingArrays.findIndex(({ id }) => id === slice.backingArray);
      const rowY = 14 + arrayIndex * 112;
      const targetX = 84 + slice.start * 48 + 21;
      const targetY = rowY + 36;
      const startX = 28 + index * 4;
      const startY = this.referenceY(index) - 5;
      const connectorY = this.referenceY(index) + 8;
      const gutterX = 8 - index * 4;
      const targetLaneY = targetY + 3;
      return 'M ' + startX + ' ' + startY + ' L ' + startX + ' ' + connectorY + ' L ' + gutterX + ' ' + connectorY + ' L ' + gutterX + ' ' + targetLaneY + ' L ' + targetX + ' ' + targetLaneY + ' L ' + targetX + ' ' + targetY + ' M ' + (targetX - 4) + ' ' + (targetY + 4) + ' L ' + targetX + ' ' + targetY + ' L ' + (targetX + 4) + ' ' + (targetY + 4);
    },
  },
};
</script>

<style scoped>
.interactive-model {
  --model-border: var(--vp-c-divider, #d7dce2);
  --model-panel: var(--vp-c-bg-soft, #f5f6f8);
  --model-text: var(--vp-c-text-1, #20242a);
  --model-muted: var(--vp-c-text-2, #656d78);
  --model-accent: var(--vp-c-brand-1, #3451b2);
  margin: 1.5rem 0;
  padding: 1rem;
  border: 1px solid var(--model-border);
  border-radius: 0.8rem;
  color: var(--model-text);
  background: var(--vp-c-bg, #fff);
}
.model-select-label { display: grid; gap: 0.35rem; font-weight: 600; }
.model-select-label select { min-height: 44px; padding: 0.55rem; border: 1px solid var(--model-border); border-radius: 0.45rem; color: inherit; background: var(--vp-c-bg, #fff); font: inherit; }
.model-controls { display: flex; flex-wrap: wrap; gap: 0.55rem; margin: 0.75rem 0; }
.model-controls button { min-height: 44px; padding: 0.55rem 0.85rem; border: 1px solid var(--model-border); border-radius: 0.45rem; color: inherit; background: var(--model-panel); font: inherit; cursor: pointer; }
.model-controls button:disabled { opacity: 0.55; cursor: default; }
.model-select-label select:focus-visible, .model-controls button:focus-visible { outline: 3px solid var(--model-accent); outline-offset: 2px; }
.model-live-status {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
.model-summary { margin: 0.6rem 0 0.2rem; }
.model-version { margin: 0; color: var(--model-muted); font-size: 0.9rem; }
.model-code { overflow-x: auto; margin: 0.75rem 0; padding: 0.75rem; border: 1px solid var(--model-border); border-radius: 0.45rem; background: var(--model-panel); }
.model-svg { display: block; width: 100%; height: auto; margin: 0.75rem 0; overflow: visible; }
.svg-label, .svg-caption, .svg-index, .svg-value { fill: var(--model-text); font-family: inherit; }
.svg-label { font-size: 16px; }
.svg-caption, .svg-index { fill: var(--model-muted); font-size: 15px; }
.svg-value { font-size: 16px; font-weight: 700; }
.svg-cell { fill: var(--model-panel); stroke: var(--model-border); stroke-width: 1.5; }
.svg-reference { fill: none; stroke: var(--model-accent); stroke-width: 1.6; }
.model-text-state { padding: 0.7rem; border-radius: 0.5rem; background: var(--model-panel); }
.model-text-state p { margin: 0.25rem 0; }
.model-text-state ul { margin: 0.4rem 0; padding-left: 1.3rem; }
.model-assumptions { color: var(--model-muted); font-size: 0.9rem; }
@media (max-width: 390px) {
  .interactive-model { padding: 0.7rem; }
  .model-controls button { flex: 1 1 9rem; }
  .model-text-state { padding: 0.55rem; }
}
</style>
