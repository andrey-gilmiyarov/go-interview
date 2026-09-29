function jobs(statuses) {
  return statuses.map((status, index) => ({
    id: `job-${index + 1}`,
    input: index + 1,
    status,
  }));
}

function worker(id, status = 'idle', jobId = null) {
  return { id, status, jobId };
}

const scenariosById = {
  'normal-completion': {
    id: 'normal-completion',
    label: 'Все задачи завершились',
    summary: 'Два worker обрабатывают четыре задачи; результат сохраняет порядок входа.',
    assumptions: 'Go 1.27.1. Это вручную заданная учебная трасса с двумя worker. Она не моделирует гарантии планировщика Go.',
    steps: [
      {
        code: 'Очередь: [job-1, job-2, job-3, job-4]\nЛимит: два worker',
        explanation: 'Четыре задачи ждут в очереди; одновременно можно выполнять не более двух.',
        view: {
          maxWorkers: 2,
          queue: ['job-1', 'job-2', 'job-3', 'job-4'],
          jobs: jobs(['queued', 'queued', 'queued', 'queued']),
          workers: [worker('W1'), worker('W2')],
          contextCancelled: false,
          joined: false,
          result: null,
        },
      },
      {
        code: 'W1 начинает job-1; W2 начинает job-2',
        explanation: 'Два worker забирают первые задачи. Остальные остаются в очереди.',
        view: {
          maxWorkers: 2,
          queue: ['job-3', 'job-4'],
          jobs: jobs(['running', 'running', 'queued', 'queued']),
          workers: [worker('W1', 'running', 'job-1'), worker('W2', 'running', 'job-2')],
          contextCancelled: false,
          joined: false,
          result: null,
        },
      },
      {
        code: 'job-1 завершен; свободный W1 начинает job-3',
        explanation: 'После завершения job-1 свободный W1 берет следующую задачу из очереди.',
        view: {
          maxWorkers: 2,
          queue: ['job-4'],
          jobs: jobs(['completed', 'running', 'running', 'queued']),
          workers: [worker('W1', 'running', 'job-3'), worker('W2', 'running', 'job-2')],
          contextCancelled: false,
          joined: false,
          result: null,
        },
      },
      {
        code: 'job-2 завершен; свободный W2 начинает job-4',
        explanation: 'Второй worker также берет следующую задачу. Выдача очереди в примере задана явно.',
        view: {
          maxWorkers: 2,
          queue: [],
          jobs: jobs(['completed', 'completed', 'running', 'running']),
          workers: [worker('W1', 'running', 'job-3'), worker('W2', 'running', 'job-4')],
          contextCancelled: false,
          joined: false,
          result: null,
        },
      },
      {
        code: 'job-3 завершен; W1 свободен',
        explanation: 'job-3 готов, W1 больше не выполняет работу.',
        view: {
          maxWorkers: 2,
          queue: [],
          jobs: jobs(['completed', 'completed', 'completed', 'running']),
          workers: [worker('W1'), worker('W2', 'running', 'job-4')],
          contextCancelled: false,
          joined: false,
          result: null,
        },
      },
      {
        code: 'job-4 завершен; все задачи завершены; worker присоединены',
        explanation: 'Все задачи завершены, worker присоединены. Результаты расположены в порядке входных значений.',
        view: {
          maxWorkers: 2,
          queue: [],
          jobs: jobs(['completed', 'completed', 'completed', 'completed']),
          workers: [worker('W1'), worker('W2')],
          contextCancelled: false,
          joined: true,
          result: [1, 2, 3, 4],
        },
      },
    ],
  },
  'cooperative-cancel': {
    id: 'cooperative-cancel',
    label: 'Отмена с ожиданием worker',
    summary: 'Очередь прекращает старт, а уже запущенные callback завершаются кооперативно.',
    assumptions: 'Go 1.27.1. Кооперативные callback проверяют context и возвращаются после отмены. Worker pool ждет их завершения; cancel не останавливает функцию принудительно.',
    steps: [
      {
        code: 'Очередь: [job-1, job-2, job-3, job-4]\nЛимит: два worker',
        explanation: 'До отмены первые две задачи еще ожидают запуска.',
        view: {
          maxWorkers: 2,
          queue: ['job-1', 'job-2', 'job-3', 'job-4'],
          jobs: jobs(['queued', 'queued', 'queued', 'queued']),
          workers: [worker('W1'), worker('W2')],
          contextCancelled: false,
          joined: false,
          result: null,
        },
      },
      {
        code: 'W1 начинает job-1; W2 начинает job-2',
        explanation: 'Два callback выполняются и должны учитывать context. Остальные задачи еще стоят в очереди.',
        view: {
          maxWorkers: 2,
          queue: ['job-3', 'job-4'],
          jobs: jobs(['running', 'running', 'queued', 'queued']),
          workers: [worker('W1', 'running', 'job-1'), worker('W2', 'running', 'job-2')],
          contextCancelled: false,
          joined: false,
          result: null,
        },
      },
      {
        code: 'Контекст отменен; задачи из очереди пропущены',
        explanation: 'Отмена не завершает callback принудительно. Новые задачи не стартуют; уже работающие пока остаются running.',
        view: {
          maxWorkers: 2,
          queue: [],
          jobs: jobs(['running', 'running', 'cancelled', 'cancelled']),
          workers: [worker('W1', 'running', 'job-1'), worker('W2', 'running', 'job-2')],
          contextCancelled: true,
          joined: false,
          result: null,
        },
      },
      {
        code: 'Кооперативные callback увидели отмену и завершились; worker присоединены',
        explanation: 'Кооперативные callback возвращаются; только после этого обе задачи отмечаются отмененными и ожидание завершается.',
        view: {
          maxWorkers: 2,
          queue: [],
          jobs: jobs(['cancelled', 'cancelled', 'cancelled', 'cancelled']),
          workers: [worker('W1'), worker('W2')],
          contextCancelled: true,
          joined: true,
          result: null,
        },
      },
    ],
  },
};

export const scenarios = Object.freeze(Object.values(scenariosById).map(({ id, label, summary }) =>
  Object.freeze({ id, label, summary }),
));

function makeState(scenarioId, stepIndex) {
  const scenario = scenariosById[scenarioId];
  if (!scenario) throw new RangeError(`Unknown worker-pool scenario: ${scenarioId}`);
  const step = scenario.steps[stepIndex];
  return {
    scenarioId,
    stepIndex,
    finished: stepIndex === scenario.steps.length - 1,
    goVersion: 'Go 1.27.1',
    assumptions: scenario.assumptions,
    code: step.code,
    explanation: step.explanation,
    view: structuredClone(step.view),
  };
}

export function createState(scenarioId) {
  return makeState(scenarioId, 0);
}

export function next(state) {
  const scenario = scenariosById[state.scenarioId];
  if (!scenario) throw new RangeError(`Unknown worker-pool scenario: ${state.scenarioId}`);
  if (state.finished) return state;
  return makeState(state.scenarioId, state.stepIndex + 1);
}

export function reset(state) {
  return createState(state.scenarioId);
}
