const scenariosById = {
  'spare-capacity': {
    id: 'spare-capacity',
    label: 'Есть свободная capacity',
    summary: 'append использует свободную ячейку того же массива.',
    assumptions: 'Go 1.27.1. В примере capacity задана явно: make([]int, 2, 3). Модель показывает backing array, а не внутреннее устройство runtime.',
    steps: [
      {
        code: 'a := make([]int, 2, 3)\na[0], a[1] = 1, 2\ns := a[:2]',
        explanation: 'Оба slice header ссылаются на массив A. У массива есть третья ячейка.',
        view: {
          backingArrays: [{ id: 'A', cells: [1, 2, 0], capacity: 3 }],
          slices: [
            { name: 'a', backingArray: 'A', start: 0, length: 2, capacity: 3 },
            { name: 's', backingArray: 'A', start: 0, length: 2, capacity: 3 },
          ],
        },
      },
      {
        code: 's = append(s, 9)',
        explanation: 'Элемент 9 занимает свободную ячейку массива A. Длина s увеличивается, а длина a остается равной двум.',
        view: {
          backingArrays: [{ id: 'A', cells: [1, 2, 9], capacity: 3 }],
          slices: [
            { name: 'a', backingArray: 'A', start: 0, length: 2, capacity: 3 },
            { name: 's', backingArray: 'A', start: 0, length: 3, capacity: 3 },
          ],
        },
      },
    ],
  },
  'full-capacity': {
    id: 'full-capacity',
    label: 'Capacity исчерпана',
    summary: 'append создает другой backing array; точный размер не задан.',
    assumptions: 'Go 1.27.1. Capacity задана явно: make([]int, 2, 2). После роста новый capacity не меньше длины slice, но его точное значение зависит от реализации и здесь намеренно не указано.',
    steps: [
      {
        code: 'a := make([]int, 2, 2)\na[0], a[1] = 1, 2\ns := a[:2]',
        explanation: 'Массив A заполнен. a и s пока ссылаются на него.',
        view: {
          backingArrays: [{ id: 'A', cells: [1, 2], capacity: 2 }],
          slices: [
            { name: 'a', backingArray: 'A', start: 0, length: 2, capacity: 2 },
            { name: 's', backingArray: 'A', start: 0, length: 2, capacity: 2 },
          ],
        },
      },
      {
        code: 's = append(s, 9)',
        explanation: 'Свободного места нет: append возвращает slice, указывающий на новый массив B. Исходный a по-прежнему ссылается на A.',
        view: {
          backingArrays: [
            { id: 'A', cells: [1, 2], capacity: 2 },
            { id: 'B', cells: [1, 2, 9], capacity: null },
          ],
          slices: [
            { name: 'a', backingArray: 'A', start: 0, length: 2, capacity: 2 },
            { name: 's', backingArray: 'B', start: 0, length: 3, capacity: null },
          ],
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
  if (!scenario) throw new RangeError(`Unknown slice scenario: ${scenarioId}`);
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
  if (!scenario) throw new RangeError(`Unknown slice scenario: ${state.scenarioId}`);
  if (state.finished) return state;
  return makeState(state.scenarioId, state.stepIndex + 1);
}

export function reset(state) {
  return createState(state.scenarioId);
}
