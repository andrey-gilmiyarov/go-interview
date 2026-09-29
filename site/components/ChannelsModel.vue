<template>
  <section class="interactive-model" aria-label="Интерактивная модель каналов Go">
    <label class="model-select-label">
      Сценарий
      <select :value="state.scenarioId" aria-label="Сценарий модели каналов" @change="selectScenario">
        <option v-for="scenario in scenarios" :key="scenario.id" :value="scenario.id">{{ scenario.label }}</option>
      </select>
    </label>

    <div class="model-controls" role="group" aria-label="Управление моделью каналов">
      <button type="button" :disabled="state.finished" @click="step">Следующий шаг</button>
      <button type="button" @click="reset">Сбросить сценарий</button>
    </div>

    <p class="model-summary">{{ currentScenario.summary }}</p>
    <p class="model-version">{{ state.goVersion }} · шаг {{ state.stepIndex + 1 }}</p>
    <p class="model-live-status" role="status" aria-live="polite" aria-atomic="true">{{ statusAnnouncement }}</p>
    <p class="trace-label">Фрагмент и операция для текущего шага.</p>
    <pre class="model-code"><code>{{ state.code }}</code></pre>

    <svg class="model-svg" viewBox="0 0 360 196" role="img" :aria-label="diagramLabel">
      <g v-for="(channel, index) in state.view.channels" :key="channel.id">
        <rect
          class="svg-card"
          :x="columnX(index) + 4"
          y="10"
          :width="columnWidth - 8"
          height="174"
          rx="8"
        />
        <text class="svg-heading" :x="columnX(index) + columnWidth / 2" y="31" text-anchor="middle">
          {{ channel.name }}{{ channel.isNil ? ' = nil' : '' }}
        </text>
        <text class="svg-caption" :x="columnX(index) + columnWidth / 2" y="47" text-anchor="middle">
          {{ channel.isNil ? 'неинициализирован' : (channel.closed ? 'закрыт' : 'открыт') }}
        </text>
        <g v-if="channel.capacity > 0">
          <g v-for="slotIndex in slotCount(channel)" :key="channel.id + '-' + slotIndex">
            <rect class="svg-buffer-cell" :x="bufferCellX(index, slotIndex - 1)" y="58" width="32" height="34" rx="4" />
            <text class="svg-buffer-value" :x="bufferCellX(index, slotIndex - 1) + 16" y="79" text-anchor="middle">
              {{ slotIndex - 1 < channel.buffer.length ? channel.buffer[slotIndex - 1] : '·' }}
            </text>
          </g>
          <text class="svg-caption" :x="columnX(index) + columnWidth / 2" y="105" text-anchor="middle">
            буфер {{ channel.buffer.length }}/{{ channel.capacity }}
          </text>
        </g>
        <text v-else class="svg-caption" :x="columnX(index) + columnWidth / 2" y="76" text-anchor="middle">
          {{ channel.isNil ? 'операции блокируются' : 'без буфера' }}
        </text>
        <text class="svg-waiter" :x="columnX(index) + 12" y="132">
          отправители: {{ channel.senders.length ? channel.senders.join(', ') : 'нет' }}
        </text>
        <text class="svg-waiter" :x="columnX(index) + 12" y="154">
          получатели: {{ channel.receivers.length ? channel.receivers.join(', ') : 'нет' }}
        </text>
      </g>
    </svg>

    <div class="model-text-state">
      <p>{{ state.explanation }}</p>
      <ul>
        <li v-for="channel in state.view.channels" :key="channel.id">
          {{ channel.name }}: {{ channel.isNil ? 'nil' : 'capacity ' + channel.capacity }};
          буфер [{{ channel.buffer.join(', ') }}];
          ожидают отправители: {{ channel.senders.join(', ') || 'нет' }};
          получатели: {{ channel.receivers.join(', ') || 'нет' }};
          {{ channel.closed ? 'закрыт' : 'открыт' }}.
        </li>
        <li v-if="state.view.received !== null">Получено сейчас: {{ state.view.received }}.</li>
        <li v-for="(receive, index) in state.view.receives" :key="'receive-' + index">
          Получение {{ index + 1 }}: value={{ receive.value }}, ok={{ receive.ok }}.
        </li>
        <li v-if="state.view.readyCasesAtSelection">
          Перед select готовы case: {{ state.view.readyCasesAtSelection.join(' и ') }};
          выбран: {{ state.view.selectedCase || 'еще не выбран' }}.
        </li>
      </ul>
      <p class="model-assumptions">{{ state.assumptions }}</p>
    </div>
  </section>
</template>

<script>
import * as model from '../models/channels.mjs';

export default {
  name: 'ChannelsModel',
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
    columnWidth() {
      return 336 / this.state.view.channels.length;
    },
    diagramLabel() {
      const summaries = this.state.view.channels.map((channel) => {
        const capacity = channel.isNil ? 'nil' : channel.capacity;
        return channel.name + ': capacity ' + capacity + ', buffer ' + (channel.buffer.join(', ') || 'пуст') +
          ', отправители ' + (channel.senders.join(', ') || 'нет') +
          ', получатели ' + (channel.receivers.join(', ') || 'нет');
      });
      if (this.state.view.selectedCase) summaries.push('выбран case ' + this.state.view.selectedCase);
      return summaries.join('. ');
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
    columnX(index) {
      return 12 + index * this.columnWidth;
    },
    slotCount(channel) {
      return Math.max(channel.capacity || 0, channel.buffer.length, 1);
    },
    bufferCellX(channelIndex, slotIndex) {
      const count = this.slotCount(this.state.view.channels[channelIndex]);
      return this.columnX(channelIndex) + (this.columnWidth - count * 36) / 2 + slotIndex * 36;
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
.trace-label { margin: 0.65rem 0 0.2rem; color: var(--model-muted); font-size: 0.9rem; }
.model-code { overflow-x: auto; margin: 0.35rem 0 0.75rem; padding: 0.75rem; border: 1px solid var(--model-border); border-radius: 0.45rem; background: var(--model-panel); }
.model-svg { display: block; width: 100%; height: auto; margin: 0.75rem 0; }
.svg-card { fill: var(--model-panel); stroke: var(--model-border); stroke-width: 1.5; }
.svg-heading, .svg-buffer-value, .svg-waiter, .svg-caption { fill: var(--model-text); font-family: inherit; }
.svg-heading { font-size: 13px; font-weight: 700; }
.svg-buffer-value { font-size: 12px; font-weight: 700; }
.svg-waiter { font-size: 9px; }
.svg-caption { fill: var(--model-muted); font-size: 10px; }
.svg-buffer-cell { fill: var(--vp-c-bg, #fff); stroke: var(--model-accent); stroke-width: 1.4; }
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
