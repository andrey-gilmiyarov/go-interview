<template>
  <section class="interactive-model" aria-label="Интерактивная модель интерфейсов Go">
    <label class="model-select-label">
      Сценарий
      <select :value="state.scenarioId" aria-label="Сценарий модели интерфейса" @change="selectScenario">
        <option v-for="scenario in scenarios" :key="scenario.id" :value="scenario.id">{{ scenario.label }}</option>
      </select>
    </label>

    <div class="model-controls" role="group" aria-label="Управление моделью интерфейса">
      <button type="button" :disabled="state.finished" @click="step">Следующий шаг</button>
      <button type="button" @click="reset">Сбросить сценарий</button>
    </div>

    <p class="model-summary">{{ currentScenario.summary }}</p>
    <p class="model-version">{{ state.goVersion }} · шаг {{ state.stepIndex + 1 }}</p>
    <p class="model-live-status" role="status" aria-live="polite" aria-atomic="true">{{ statusAnnouncement }}</p>
    <pre class="model-code"><code>{{ state.code }}</code></pre>

    <svg class="model-svg" viewBox="0 0 360 148" role="img" :aria-label="diagramLabel">
      <rect class="svg-card" x="12" y="18" width="106" height="108" rx="8" />
      <text class="svg-heading" x="65" y="40" text-anchor="middle">указатель p</text>
      <text class="svg-value" x="65" y="69" text-anchor="middle">
        {{ state.view.pointer ? state.view.pointer.type : 'нет' }}
      </text>
      <text class="svg-caption" x="65" y="94" text-anchor="middle">
        {{ state.view.pointer ? (state.view.pointer.isNil ? 'nil' : 'значение') : 'не задан' }}
      </text>

      <path class="svg-connector" d="M 118 71 H 143" />
      <rect class="svg-card" x="145" y="18" width="203" height="108" rx="8" />
      <text class="svg-heading" x="246" y="40" text-anchor="middle">интерфейс err</text>
      <text class="svg-caption" x="158" y="68">динамический тип</text>
      <text class="svg-value" x="158" y="84">{{ state.view.interface.dynamicType || 'отсутствует' }}</text>
      <text class="svg-caption" x="158" y="103">значение</text>
      <text class="svg-value svg-small" x="158" y="118">
        {{ state.view.interface.value === null ? 'nil' : state.view.interface.value }}
      </text>
    </svg>

    <div class="model-text-state">
      <p>{{ state.explanation }}</p>
      <ul>
        <li>
          Указатель p:
          {{ state.view.pointer ? `${state.view.pointer.type}, ${state.view.pointer.isNil ? 'nil' : 'не nil'}` : 'не объявлен в этом примере' }}.
        </li>
        <li>
          err: динамический тип {{ state.view.interface.dynamicType || 'отсутствует' }};
          значение {{ state.view.interface.value === null ? 'nil' : state.view.interface.value }}.
        </li>
        <li>err == nil: {{ state.view.nilComparison ? 'true' : 'false' }}.</li>
      </ul>
      <p class="model-assumptions">{{ state.assumptions }}</p>
    </div>
  </section>
</template>

<script>
import * as model from '../models/interfaces.mjs';

export default {
  name: 'InterfaceModel',
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
    diagramLabel() {
      const type = this.state.view.interface.dynamicType || 'динамический тип отсутствует';
      const value = this.state.view.interface.value === null ? 'nil-значение' : this.state.view.interface.value;
      return `Указатель ${this.state.view.pointer?.type || 'не задан'}; интерфейс err: ${type}, ${value}; err == nil: ${this.state.view.nilComparison}`;
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
.svg-heading { font-size: 13px; font-weight: 700; }
.svg-value { font-size: 12px; font-weight: 700; }
.svg-small { font-size: 10px; }
.svg-caption { fill: var(--model-muted); font-size: 10px; }
.svg-connector { stroke: var(--model-accent); stroke-width: 2; }
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
