function channel(id, capacity, buffer = [], overrides = {}) {
  return {
    id,
    name: id,
    capacity,
    buffer,
    senders: [],
    receivers: [],
    closed: false,
    isNil: false,
    ...overrides,
  };
}

const scenariosById = {
  unbuffered: {
    id: 'unbuffered',
    label: 'Небуферизованный канал',
    summary: 'Отправитель ждет получателя для rendezvous.',
    assumptions: 'Go 1.27.1. Две отдельные горутины: Gsender блокируется на отправке, затем Greceiver выполняет получение.',
    steps: [
      {
        code: 'ch := make(chan int)\n// Gsender: ch <- 7\n// Greceiver: <-ch',
        explanation: 'Небуферизованный канал не хранит значения между отправкой и получением.',
        view: { channels: [channel('ch', 0)], received: null, receives: [], selectedCase: null },
      },
      {
        code: '// Gsender: ch <- 7',
        explanation: 'Пока получателя нет, отправка блокируется. Значение не попадает в буфер.',
        view: { channels: [channel('ch', 0, [], { senders: ['Gsender: 7'] })], received: null, receives: [], selectedCase: null },
      },
      {
        code: '// Greceiver: value := <-ch',
        explanation: 'Получатель встречается с ожидающей отправкой. Значение передается напрямую, без буфера.',
        view: {
          channels: [channel('ch', 0)],
          received: 7,
          receives: [{ value: 7, ok: true }],
          selectedCase: null,
        },
      },
    ],
  },
  buffered: {
    id: 'buffered',
    label: 'Буферизованный канал',
    summary: 'Отправка помещает значение в буфер, пока он не заполнен.',
    assumptions: 'Go 1.27.1. Capacity канала явно равна двум; показана одна отправка и одно получение.',
    steps: [
      {
        code: 'ch := make(chan int, 2)',
        explanation: 'Канал может временно хранить до двух значений.',
        view: { channels: [channel('ch', 2)], received: null, receives: [], selectedCase: null },
      },
      {
        code: '// Gsender: ch <- 7',
        explanation: 'В буфере есть место, поэтому отправка завершается и значение 7 появляется в буфере.',
        view: { channels: [channel('ch', 2, [7])], received: null, receives: [], selectedCase: null },
      },
      {
        code: '// Greceiver: value := <-ch',
        explanation: 'Получение забирает самое раннее значение из буфера.',
        view: { channels: [channel('ch', 2)], received: 7, receives: [{ value: 7, ok: true }], selectedCase: null },
      },
    ],
  },
  'buffered-full': {
    id: 'buffered-full',
    label: 'Полный буфер и ожидающая отправка',
    summary: 'Получение освобождает слот, после чего ожидающая отправка может продолжиться.',
    assumptions: 'Go 1.27.1. Capacity равна одному. Gsender пытается отправить во время заполненного буфера; получение из другой горутины освобождает место.',
    steps: [
      {
        code: 'ch := make(chan int, 1)\nch <- 7',
        explanation: 'Единственный слот занят значением 7.',
        view: { channels: [channel('ch', 1, [7])], received: null, receives: [], selectedCase: null },
      },
      {
        code: '// Gsender: ch <- 8',
        explanation: 'Буфер полон, поэтому отправитель ждет; значение 8 пока не добавлено.',
        view: {
          channels: [channel('ch', 1, [7], { senders: ['Gsender: 8'] })],
          received: null,
          receives: [],
          selectedCase: null,
        },
      },
      {
        code: '// Greceiver: value := <-ch',
        explanation: 'Получатель забирает 7 и освобождает слот. Ожидающая отправка может поместить 8 в буфер.',
        view: {
          channels: [channel('ch', 1, [8])],
          received: 7,
          receives: [{ value: 7, ok: true }],
          selectedCase: null,
        },
      },
      {
        code: '// Greceiver: value := <-ch',
        explanation: 'Следующее получение забирает 8.',
        view: {
          channels: [channel('ch', 1)],
          received: 8,
          receives: [{ value: 7, ok: true }, { value: 8, ok: true }],
          selectedCase: null,
        },
      },
    ],
  },
  'closed-channel': {
    id: 'closed-channel',
    label: 'Закрытый канал',
    summary: 'Сначала читаются буферизированные значения, затем ok становится false.',
    assumptions: 'Go 1.27.1. Канал закрывает отправитель; повторное чтение из опустошенного закрытого канала возвращает zero value и ok=false.',
    steps: [
      {
        code: 'ch := make(chan int, 1)\nch <- 7',
        explanation: 'Перед закрытием в буфере находится одно значение.',
        view: { channels: [channel('ch', 1, [7])], received: null, receives: [], selectedCase: null },
      },
      {
        code: 'close(ch)',
        explanation: 'Закрытие запрещает новые отправки, но не удаляет значение, уже находящееся в буфере.',
        view: { channels: [channel('ch', 1, [7], { closed: true })], received: null, receives: [], selectedCase: null },
      },
      {
        code: 'value, ok := <-ch',
        explanation: 'Из закрытого канала можно дочитать сохраненное значение; для него ok=true.',
        view: {
          channels: [channel('ch', 1, [], { closed: true })],
          received: 7,
          receives: [{ value: 7, ok: true }],
          selectedCase: null,
        },
      },
      {
        code: 'value, ok = <-ch',
        explanation: 'Когда буфер пуст, чтение закрытого канала немедленно возвращает zero value и ok=false.',
        view: {
          channels: [channel('ch', 1, [], { closed: true })],
          received: 0,
          receives: [{ value: 7, ok: true }, { value: 0, ok: false }],
          selectedCase: null,
        },
      },
    ],
  },
  'nil-channel': {
    id: 'nil-channel',
    label: 'Nil-канал',
    summary: 'Отправка и получение по nil-каналу блокируются навсегда.',
    assumptions: 'Go 1.27.1. Gsender и Greceiver — разные горутины; обе операции остаются заблокированными на одном nil-канале. Context и другие способы разблокировки не показаны.',
    steps: [
      {
        code: 'var ch chan int',
        explanation: 'Нулевая channel-переменная равна nil и не имеет буфера.',
        view: { channels: [channel('ch', null, [], { isNil: true })], received: null, receives: [], selectedCase: null },
      },
      {
        code: 'ch <- 7',
        explanation: 'Отправка в nil-канал блокируется навсегда.',
        view: {
          channels: [channel('ch', null, [], { isNil: true, senders: ['Gsender: 7'] })],
          received: null,
          receives: [],
          selectedCase: null,
        },
      },
      {
        code: 'value := <-ch',
        explanation: 'Получение из nil-канала тоже блокируется; ни одна сторона не может встретиться.',
        view: {
          channels: [channel('ch', null, [], { isNil: true, senders: ['Gsender: 7'], receivers: ['Greceiver'] })],
          received: null,
          receives: [],
          selectedCase: null,
        },
      },
    ],
  },
  'select-left': {
    id: 'select-left',
    label: 'select: выбран left',
    summary: 'Оба case готовы; эта допустимая трасса выбирает left.',
    assumptions: 'Go 1.27.1. Оба case готовы. Выбор case не гарантирован по порядку или приоритету; здесь показана одна допустимая трасса, а select-right показывает другую.',
    steps: [
      {
        code: 'select {\ncase v := <-left: println(v)\ncase v := <-right: println(v)\n}',
        explanation: 'В буферах обоих каналов есть значение, поэтому готовы оба receive-case.',
        view: {
          channels: [channel('left', 1, [1]), channel('right', 1, [2])],
          received: null,
          receives: [],
          selectedCase: null,
          readyCasesAtSelection: ['left', 'right'],
        },
      },
      {
        code: '// эта трасса выбрала left',
        explanation: 'Один из готовых case выбран. Другой остается готовым, но порядок выбора не задает приоритет.',
        view: {
          channels: [channel('left', 1), channel('right', 1, [2])],
          received: 1,
          receives: [{ value: 1, ok: true }],
          selectedCase: 'left',
          readyCasesAtSelection: ['left', 'right'],
        },
      },
    ],
  },
  'select-right': {
    id: 'select-right',
    label: 'select: выбран right',
    summary: 'Та же готовность; другая допустимая трасса выбирает right.',
    assumptions: 'Go 1.27.1. Оба case готовы. Выбор case не гарантирован по порядку или приоритету; здесь показана одна допустимая трасса, а select-left показывает другую.',
    steps: [
      {
        code: 'select {\ncase v := <-left: println(v)\ncase v := <-right: println(v)\n}',
        explanation: 'В буферах обоих каналов есть значение, поэтому готовы оба receive-case.',
        view: {
          channels: [channel('left', 1, [1]), channel('right', 1, [2])],
          received: null,
          receives: [],
          selectedCase: null,
          readyCasesAtSelection: ['left', 'right'],
        },
      },
      {
        code: '// эта трасса выбрала right',
        explanation: 'Другой готовый case также может быть выбран. Модель показывает допустимый исход, а не прогноз планировщика.',
        view: {
          channels: [channel('left', 1, [1]), channel('right', 1)],
          received: 2,
          receives: [{ value: 2, ok: true }],
          selectedCase: 'right',
          readyCasesAtSelection: ['left', 'right'],
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
  if (!scenario) throw new RangeError(`Unknown channel scenario: ${scenarioId}`);
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
  if (!scenario) throw new RangeError(`Unknown channel scenario: ${state.scenarioId}`);
  if (state.finished) return state;
  return makeState(state.scenarioId, state.stepIndex + 1);
}

export function reset(state) {
  return createState(state.scenarioId);
}
