const scenariosById = {
  'nil-interface': {
    id: 'nil-interface',
    label: 'Nil interface',
    summary: 'У интерфейса нет ни динамического типа, ни значения.',
    assumptions: 'Go 1.27.1. Показана пара «динамический тип — динамическое значение» как языковая модель интерфейса.',
    steps: [
      {
        code: 'var err error = nil',
        explanation: 'Пустое значение интерфейса равно nil: динамический тип и динамическое значение отсутствуют.',
        view: {
          pointer: null,
          interface: { dynamicType: null, value: null, isNil: true },
          nilComparison: true,
        },
      },
    ],
  },
  'typed-nil': {
    id: 'typed-nil',
    label: 'Typed nil',
    summary: 'Nil-указатель после присваивания хранит динамический тип.',
    assumptions: 'Go 1.27.1. Не вызываем метод на nil-указателе; сравниваем только интерфейс с nil.',
    steps: [
      {
        code: 'var p *MyError = nil\nvar err error',
        explanation: 'Указатель p равен nil. Пока в err ничего не записано, сам интерфейс тоже равен nil.',
        view: {
          pointer: { type: '*MyError', isNil: true },
          interface: { dynamicType: null, value: null, isNil: true },
          nilComparison: true,
        },
      },
      {
        code: 'err = p',
        explanation: 'После присваивания интерфейс содержит динамический тип *MyError и nil-указатель. Поэтому err != nil.',
        view: {
          pointer: { type: '*MyError', isNil: true },
          interface: { dynamicType: '*MyError', value: null, isNil: false },
          nilComparison: false,
        },
      },
    ],
  },
  'concrete-pointer': {
    id: 'concrete-pointer',
    label: 'Ненулевой указатель',
    summary: 'Интерфейс содержит тип *MyError и конкретное значение.',
    assumptions: 'Go 1.27.1. Пример показывает присваивание значения *MyError интерфейсу error.',
    steps: [
      {
        code: 'err := error(&MyError{message: "boom"})',
        explanation: 'У интерфейса есть динамический тип и ненулевое динамическое значение, поэтому сравнение с nil дает false.',
        view: {
          pointer: { type: '*MyError', isNil: false },
          interface: { dynamicType: '*MyError', value: '&MyError{message: "boom"}', isNil: false },
          nilComparison: false,
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
  if (!scenario) throw new RangeError(`Unknown interface scenario: ${scenarioId}`);
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
  if (!scenario) throw new RangeError(`Unknown interface scenario: ${state.scenarioId}`);
  if (state.finished) return state;
  return makeState(state.scenarioId, state.stepIndex + 1);
}

export function reset(state) {
  return createState(state.scenarioId);
}
