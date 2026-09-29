<template>
  <section class="interactive-model" aria-label="Интерактивная модель defer в Go">
    <label class="model-select-label">
      Сценарий
      <select :value="state.scenarioId" aria-label="Сценарий модели defer" @change="selectScenario">
        <option v-for="scenario in scenarios" :key="scenario.id" :value="scenario.id">{{ scenario.label }}</option>
      </select>
    </label>

    <div class="model-controls" role="group" aria-label="Управление моделью defer">
      <button type="button" :disabled="state.finished" @click="step">Следующий шаг</button>
      <button type="button" @click="reset">Сбросить сценарий</button>
    </div>

    <p class="model-summary">{{ currentScenario.summary }}</p>
    <p class="model-version">{{ state.goVersion }} · шаг {{ state.stepIndex + 1 }}</p>
    <p class="model-live-status" role="status" aria-live="polite" aria-atomic="true">{{ statusAnnouncement }}</p>
    <pre class="model-code"><code>{{ state.code }}</code></pre>

    <svg class="model-svg" :viewBox="`0 0 360 ${diagramHeight}`" role="img" :aria-label="diagramLabel">
      <text class="svg-heading" x="14" y="22">Текущее значение</text>
      <rect class="svg-card" x="14" y="32" width="332" height="36" rx="6" />
      <text class="svg-value" x="28" y="55">{{ resultLabel }}</text>

      <text class="svg-heading" x="14" y="91">Стек defer · сверху выполняется последняя регистрация</text>
      <g v-for="(entry, index) in state.view.stack" :key="`${entry.kind}-${index}`">
        <rect
          class="svg-card"
          x="30"
          :y="104 + (state.view.stack.length - index - 1) * 39"
          width="316"
          height="31"
          rx="5"
        />
        <text
          class="svg-value"
          x="42"
          :y="124 + (state.view.stack.length - index - 1) * 39"
        >
          {{ entry.kind === 'evaluated-call' ? `вызов: ${entry.expression}, сохранено ${entry.savedArgument}` : `замыкание: ${entry.expression}` }}
        </text>
      </g>
      <text class="svg-caption" x="14" :y="116 + state.view.stack.length * 39">
        Записанные события: {{ state.view.events.length ? state.view.events.join(' → ') : 'пока нет' }}
      </text>
    </svg>

    <div class="model-text-state">
      <p>{{ state.explanation }}</p>
      <p>Стек снизу вверх: {{ stackText || 'пуст' }}.</p>
      <p>События: {{ state.view.events.length ? state.view.events.join(' → ') : 'пока нет' }}.</p>
      <p class="model-assumptions">{{ state.assumptions }}</p>
    </div>
  </section>
</template>

<script>
import * as model from '../models/defer.mjs';

export default {
  name: 'DeferModel',
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
    resultLabel() {
      if (Object.hasOwn(this.state.view, 'x')) return `x = ${this.state.view.x}`;
      return `n = ${this.state.view.result === null ? 'ещё не присвоен' : this.state.view.result}`;
    },
    stackText() {
      return this.state.view.stack.map((entry) => entry.kind === 'evaluated-call'
        ? `${entry.expression} (аргумент ${entry.savedArgument} сохранен)`
        : `${entry.expression} (замыкание)`).join(' → ');
    },
    diagramHeight() {
      return 130 + this.state.view.stack.length * 39;
    },
    diagramLabel() {
      const stack = this.stackText || 'стек defer пуст';
      const events = this.state.view.events.join(', ') || 'событий пока нет';
      return `${this.resultLabel}; ${stack}; события: ${events}`;
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
.model-svg { display: block; width: 100%; height: auto; margin: 0.75rem 0; }
.svg-card { fill: var(--model-panel); stroke: var(--model-border); stroke-width: 1.5; }
.svg-heading, .svg-value, .svg-caption { fill: var(--model-text); font-family: inherit; }
.svg-heading { font-size: 11px; font-weight: 700; }
.svg-value { font-size: 11px; }
.svg-caption { fill: var(--model-muted); font-size: 10px; }
.model-text-state { padding: 0.7rem; border-radius: 0.5rem; background: var(--model-panel); }
.model-text-state p { margin: 0.25rem 0; }
.model-assumptions { color: var(--model-muted); font-size: 0.9rem; }
@media (max-width: 390px) {
  .interactive-model { padding: 0.7rem; }
  .model-controls button { flex: 1 1 9rem; }
  .model-text-state { padding: 0.55rem; }
}
</style>
