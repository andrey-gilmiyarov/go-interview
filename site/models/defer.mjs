const scenariosById = {
  'argument-evaluation': {
    id: 'argument-evaluation',
    label: 'Аргумент и замыкание',
    summary: 'Аргумент defer вычисляется при регистрации, вызовы идут в обратном порядке.',
    assumptions: 'Go 1.27.1. Функция log только записывает переданное значение; паники нет.',
    steps: [
      {
        code: 'x := 1',
        explanation: 'Локальная переменная x равна 1.',
        view: { x: 1, stack: [], events: [] },
      },
      {
        code: 'defer log(x)\ndefer func() { log(x) }()',
        explanation: 'Первый defer сразу сохраняет аргумент 1. Замыкание сохраняет доступ к переменной x, которую прочитает при выполнении.',
        view: {
          x: 1,
          stack: [
            { kind: 'evaluated-call', expression: 'log(x)', savedArgument: 1 },
            { kind: 'closure', expression: 'log(x)', savedArgument: null },
          ],
          events: [],
        },
      },
      {
        code: 'x = 2',
        explanation: 'Значение x меняется. Отложенный аргумент уже равен 1, а замыкание при вызове увидит 2.',
        view: {
          x: 2,
          stack: [
            { kind: 'evaluated-call', expression: 'log(x)', savedArgument: 1 },
            { kind: 'closure', expression: 'log(x)', savedArgument: null },
          ],
          events: [],
        },
      },
      {
        code: '// выход из функции: запускается верхний defer',
        explanation: 'Последний зарегистрированный defer запускается первым. Замыкание читает текущее x и записывает 2.',
        view: {
          x: 2,
          stack: [{ kind: 'evaluated-call', expression: 'log(x)', savedArgument: 1 }],
          events: [2],
        },
      },
      {
        code: '// запускается первый defer',
        explanation: 'Обычный вызов defer использует аргумент, вычисленный при регистрации: записывается 1.',
        view: { x: 2, stack: [], events: [2, 1] },
      },
    ],
  },
  'named-result': {
    id: 'named-result',
    label: 'Именованный результат',
    summary: 'Отложенная функция может изменить именованный результат до возврата.',
    assumptions: 'Go 1.27.1. Отложенная функция выполняется после присваивания результата и до фактического возврата из функции.',
    steps: [
      {
        code: 'func f() (n int) {',
        explanation: 'При входе в функцию именованная переменная n типа int уже равна своему нулевому значению: 0.',
        view: { result: 0, stack: [], events: [] },
      },
      {
        code: 'defer func() { n++ }()',
        explanation: 'Регистрируется замыкание; значение именованного результата пока остается равным 0.',
        view: { result: 0, stack: [{ kind: 'closure', expression: 'n++' }], events: [] },
      },
      {
        code: 'return 4',
        explanation: 'Сначала выражение return присваивает 4 переменной n. Затем выполняются отложенные функции.',
        view: { result: 4, stack: [{ kind: 'closure', expression: 'n++' }], events: [] },
      },
      {
        code: '// defer: n++',
        explanation: 'Замыкание увеличивает уже присвоенный именованный результат до 5.',
        view: { result: 5, stack: [], events: ['deferred increment: 4 -> 5'] },
      },
      {
        code: '// возврат n',
        explanation: 'После завершения defer функция возвращает актуальное значение именованного результата: 5.',
        view: { result: 5, stack: [], events: ['deferred increment: 4 -> 5'] },
      },
    ],
  },
};

export const scenarios = Object.freeze(Object.values(scenariosById).map(({ id, label, summary }) =>
  Object.freeze({ id, label, summary }),
));

function makeState(scenarioId, stepIndex) {
  const scenario = scenariosById[scenarioId];
  if (!scenario) throw new RangeError(`Unknown defer scenario: ${scenarioId}`);
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
  if (!scenario) throw new RangeError(`Unknown defer scenario: ${state.scenarioId}`);
  if (state.finished) return state;
  return makeState(state.scenarioId, state.stepIndex + 1);
}

export function reset(state) {
  return createState(state.scenarioId);
}
