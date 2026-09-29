<template>
  <section class="interactive-model" aria-label="Интерактивная модель worker pool в Go">
    <label class="model-select-label">
      Сценарий
      <select :value="state.scenarioId" aria-label="Сценарий модели worker pool" @change="selectScenario">
        <option v-for="scenario in scenarios" :key="scenario.id" :value="scenario.id">{{ scenario.label }}</option>
      </select>
    </label>

    <div class="model-controls" role="group" aria-label="Управление моделью worker pool">
      <button type="button" :disabled="state.finished" @click="step">Следующий шаг</button>
      <button type="button" @click="reset">Сбросить сценарий</button>
    </div>

    <p class="model-summary">{{ currentScenario.summary }}</p>
    <p class="model-version">{{ state.goVersion }} · шаг {{ state.stepIndex + 1 }}</p>
    <p class="model-live-status" role="status" aria-live="polite" aria-atomic="true">{{ statusAnnouncement }}</p>
    <p class="trace-label">Учебная трасса состояния; это не фрагмент кода и не прогноз планировщика Go.</p>
    <div class="model-trace">{{ state.code }}</div>

    <svg class="model-svg" viewBox="0 0 360 238" role="img" :aria-label="diagramLabel">
      <text class="svg-heading" x="14" y="19">Worker · максимум {{ state.view.maxWorkers }}</text>
      <g v-for="(worker, index) in state.view.workers" :key="worker.id">
        <rect
          class="worker-card"
          :class="'worker-' + worker.status"
          :x="14 + index * 168"
          y="28"
          width="158"
          height="48"
          rx="7"
        />
        <text class="svg-worker-name" :x="26 + index * 168" y="48">{{ worker.id }} · {{ statusLabel(worker.status) }}</text>
        <text class="svg-worker-job" :x="26 + index * 168" y="66">{{ worker.jobId || 'без задачи' }}</text>
      </g>

      <text class="svg-heading" x="14" y="99">Задачи</text>
      <g v-for="(job, index) in state.view.jobs" :key="job.id">
        <rect
          class="job-row"
          :class="'job-' + job.status"
          x="14"
          :y="106 + index * 29"
          width="332"
          height="24"
          rx="5"
        />
        <text class="svg-job-text" x="26" :y="123 + index * 29">
          {{ job.id }} · {{ statusLabel(job.status) }}<template v-if="workerFor(job.id)"> · {{ workerFor(job.id) }}</template>
        </text>
      </g>
    </svg>

    <div class="model-text-state">
      <p>{{ state.explanation }}</p>
      <ul>
        <li>Очередь: {{ state.view.queue.join(', ') || 'пуста' }}.</li>
        <li v-for="job in state.view.jobs" :key="job.id">
          {{ job.id }}: {{ statusLabel(job.status) }}<template v-if="workerFor(job.id)">, выполняет {{ workerFor(job.id) }}</template>.
        </li>
        <li>Контекст: {{ state.view.contextCancelled ? 'отменен' : 'активен' }}.</li>
        <li>Ожидание worker завершено: {{ state.view.joined ? 'да' : 'нет' }}.</li>
        <li>Результат: {{ state.view.result === null ? 'не публикуется' : state.view.result.join(', ') }}.</li>
      </ul>
      <p class="model-assumptions">{{ state.assumptions }}</p>
    </div>
  </section>
</template>

<script>
import * as model from '../models/worker-pool.mjs';

export default {
  name: 'WorkerPoolModel',
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
      const workers = this.state.view.workers.map((worker) => {
        return worker.id + ': ' + this.statusLabel(worker.status) + (worker.jobId ? ', ' + worker.jobId : '');
      });
      const jobs = this.state.view.jobs.map((job) => job.id + ': ' + this.statusLabel(job.status));
      return 'Очередь: ' + (this.state.view.queue.join(', ') || 'пустая') + '. ' + workers.join('. ') + '. ' + jobs.join('. ');
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
    statusLabel(status) {
      return {
        queued: 'в очереди',
        running: 'выполняется',
        completed: 'завершена',
        cancelled: 'отменена / пропущена',
        idle: 'свободен',
      }[status] || status;
    },
    workerFor(jobId) {
      const worker = this.state.view.workers.find(({ jobId: runningJob }) => runningJob === jobId);
      return worker ? worker.id : '';
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
.model-trace { white-space: pre-wrap; margin: 0.35rem 0 0.75rem; padding: 0.75rem; border: 1px solid var(--model-border); border-radius: 0.45rem; background: var(--model-panel); }
.model-svg { display: block; width: 100%; height: auto; margin: 0.75rem 0; }
.svg-heading, .svg-worker-name, .svg-worker-job, .svg-job-text { fill: var(--model-text); font-family: inherit; }
.svg-heading { font-size: 12px; font-weight: 700; }
.svg-worker-name, .svg-job-text { font-size: 10px; font-weight: 600; }
.svg-worker-job { fill: var(--model-muted); font-size: 9px; }
.worker-card, .job-row { stroke-width: 1.2; }
.worker-idle { fill: var(--model-panel); stroke: var(--model-border); }
.worker-running { fill: var(--vp-c-brand-soft, #e9edff); stroke: var(--model-accent); }
.job-queued { fill: var(--model-panel); stroke: var(--model-border); }
.job-running { fill: var(--vp-c-brand-soft, #e9edff); stroke: var(--model-accent); }
.job-completed { fill: var(--vp-c-green-soft, #e8f5e9); stroke: var(--vp-c-green-1, #278345); }
.job-cancelled { fill: var(--vp-c-warning-soft, #fff4df); stroke: var(--vp-c-warning-1, #9a6700); }
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
