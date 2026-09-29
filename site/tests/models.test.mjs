import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

import * as slices from '../models/slices.mjs';
import * as interfaces from '../models/interfaces.mjs';
import * as defer from '../models/defer.mjs';
import * as channels from '../models/channels.mjs';
import * as workers from '../models/worker-pool.mjs';

function advance(model, state, count = 1) {
  let current = state;
  for (let i = 0; i < count; i += 1) current = model.next(current);
  return current;
}

function finish(model, state) {
  let current = state;
  let guard = 20;
  while (!current.finished && guard > 0) {
    current = model.next(current);
    guard -= 1;
  }
  assert.ok(current.finished, 'scenario should have a finite terminal state');
  return current;
}

function copy(value) {
  return structuredClone(value);
}

test('slice scenarios distinguish shared backing storage from append reallocation', () => {
  for (const scenario of slices.scenarios) {
    let state = slices.createState(scenario.id);
    while (true) {
      for (const slice of state.view.slices) {
        if (slice.capacity !== null) assert.ok(slice.length <= slice.capacity);
      }
      if (state.finished) break;
      const nextState = slices.next(state);
      for (const before of state.view.slices) {
        const after = nextState.view.slices.find(({ name }) => name === before.name);
        if (before.backingArray === after.backingArray) assert.equal(after.capacity, before.capacity);
      }
      state = nextState;
    }
  }

  assert.deepEqual(slices.scenarios.map(({ id }) => id), ['spare-capacity', 'full-capacity']);

  const sharedInitial = slices.createState('spare-capacity');
  assert.equal(sharedInitial.view.backingArrays[0].capacity, 3);
  assert.deepEqual(sharedInitial.view.backingArrays[0].cells, [1, 2, 0]);
  const sharedBefore = copy(sharedInitial);
  const sharedAfter = slices.next(sharedInitial);
  assert.deepEqual(sharedInitial, sharedBefore, 'next must leave the prior state untouched');
  assert.equal(sharedAfter.view.slices.find(({ name }) => name === 'a').backingArray, 'A');
  assert.equal(sharedAfter.view.slices.find(({ name }) => name === 's').backingArray, 'A');
  assert.equal(sharedAfter.view.slices.find(({ name }) => name === 'a').length, 2);
  assert.equal(sharedAfter.view.backingArrays[0].cells[2], 9);
  assert.equal(sharedAfter.view.slices.find(({ name }) => name === 's').length, 3);
  assert.equal(sharedAfter.view.slices.find(({ name }) => name === 's').capacity, 3);

  const detached = finish(slices, slices.createState('full-capacity'));
  assert.equal(detached.view.slices.find(({ name }) => name === 'a').backingArray, 'A');
  assert.equal(detached.view.slices.find(({ name }) => name === 's').backingArray, 'B');
  assert.deepEqual(detached.view.backingArrays.find(({ id }) => id === 'A').cells, [1, 2]);
  assert.deepEqual(detached.view.backingArrays.find(({ id }) => id === 'B').cells, [1, 2, 9]);
  assert.equal(detached.view.backingArrays.find(({ id }) => id === 'B').capacity, null,
    'the model must not claim an implementation-specific growth capacity');
  assert.equal(slices.next(detached), detached, 'next stays stable after the final step');

  const reset = slices.reset(detached);
  assert.equal(reset.scenarioId, 'full-capacity');
  assert.equal(reset.stepIndex, 0);
  assert.deepEqual(reset.view.backingArrays[0].cells, [1, 2]);
});

test('interface scenarios show nil interface, typed nil, and a concrete pointer separately', () => {
  assert.deepEqual(interfaces.scenarios.map(({ id }) => id), ['nil-interface', 'typed-nil', 'concrete-pointer']);

  const nilInterface = finish(interfaces, interfaces.createState('nil-interface'));
  assert.equal(nilInterface.view.interface.dynamicType, null);
  assert.equal(nilInterface.view.interface.value, null);
  assert.equal(nilInterface.view.interface.isNil, true);

  const typedNil = advance(interfaces, interfaces.createState('typed-nil'));
  assert.deepEqual(typedNil.view.pointer, { type: '*MyError', isNil: true });
  assert.equal(typedNil.view.interface.dynamicType, '*MyError');
  assert.equal(typedNil.view.interface.value, null);
  assert.equal(typedNil.view.interface.isNil, false);
  assert.equal(typedNil.view.nilComparison, false);

  const concrete = finish(interfaces, interfaces.createState('concrete-pointer'));
  assert.equal(concrete.view.interface.dynamicType, '*MyError');
  assert.equal(concrete.view.interface.value, '&MyError{message: "boom"}');
  assert.equal(concrete.view.interface.isNil, false);
});

test('defer scenarios separate argument evaluation, LIFO execution, and named results', () => {
  const namedResultEntry = defer.createState('named-result');
  assert.equal(namedResultEntry.view.result, 0, 'named result starts at its int zero value on function entry');
  const deferRegistered = defer.next(namedResultEntry);
  assert.equal(deferRegistered.view.result, 0, 'registering defer does not unset the named result');
  assert.match(deferRegistered.explanation, /равным 0/);
  assert.deepEqual(defer.scenarios.map(({ id }) => id), ['argument-evaluation', 'named-result']);

  const beforeMutation = advance(defer, defer.createState('argument-evaluation'), 2);
  assert.equal(beforeMutation.view.x, 2);
  assert.deepEqual(beforeMutation.view.stack.map(({ kind }) => kind), ['evaluated-call', 'closure']);
  const afterFirstDeferred = defer.next(beforeMutation);
  assert.deepEqual(afterFirstDeferred.view.events, [2]);
  assert.equal(afterFirstDeferred.view.stack.length, 1);
  const finalOrder = finish(defer, afterFirstDeferred);
  assert.deepEqual(finalOrder.view.events, [2, 1]);

  const namedResult = finish(defer, defer.createState('named-result'));
  assert.equal(namedResult.view.result, 5);
  assert.deepEqual(namedResult.view.events, ['deferred increment: 4 -> 5']);
  assert.equal(defer.next(namedResult), namedResult);
});

test('channel scenarios expose buffer and waiters for rendezvous, close, nil, and both ready select cases', () => {
  assert.deepEqual(channels.scenarios.map(({ id }) => id), [
    'unbuffered', 'buffered', 'buffered-full', 'closed-channel', 'nil-channel', 'select-left', 'select-right',
  ]);

  const waitingSender = advance(channels, channels.createState('unbuffered'));
  assert.equal(waitingSender.view.channels[0].capacity, 0);
  assert.deepEqual(waitingSender.view.channels[0].senders, ['Gsender: 7']);
  const rendezvous = finish(channels, waitingSender);
  assert.deepEqual(rendezvous.view.channels[0].senders, []);
  assert.deepEqual(rendezvous.view.channels[0].receivers, []);
  assert.deepEqual(rendezvous.view.channels[0].buffer, []);
  assert.equal(rendezvous.view.received, 7);

  const buffered = advance(channels, channels.createState('buffered'));
  assert.equal(buffered.view.channels[0].capacity, 2);
  assert.deepEqual(buffered.view.channels[0].buffer, [7]);
  const bufferedReceive = finish(channels, buffered);
  assert.deepEqual(bufferedReceive.view.channels[0].buffer, []);
  assert.equal(bufferedReceive.view.received, 7);

  const fullSenderWaiting = advance(channels, channels.createState('buffered-full'));
  assert.deepEqual(fullSenderWaiting.view.channels[0].buffer, [7]);
  assert.deepEqual(fullSenderWaiting.view.channels[0].senders, ['Gsender: 8']);
  const slotFreed = channels.next(fullSenderWaiting);
  assert.equal(slotFreed.view.received, 7);
  assert.deepEqual(slotFreed.view.channels[0].senders, []);
  assert.deepEqual(slotFreed.view.channels[0].buffer, [8]);
  const senderCompleted = finish(channels, slotFreed);
  assert.deepEqual(senderCompleted.view.receives.map(({ value }) => value), [7, 8]);

  const closed = finish(channels, channels.createState('closed-channel'));
  assert.equal(closed.view.channels[0].closed, true);
  assert.deepEqual(closed.view.receives, [{ value: 7, ok: true }, { value: 0, ok: false }]);
  assert.deepEqual(closed.view.channels[0].buffer, []);

  const nilChannel = advance(channels, channels.createState('nil-channel'), 2);
  assert.equal(nilChannel.view.channels[0].isNil, true);
  assert.deepEqual(nilChannel.view.channels[0].senders, ['Gsender: 7']);
  assert.deepEqual(nilChannel.view.channels[0].receivers, ['Greceiver']);
  assert.match(nilChannel.assumptions, /Gsender.*Greceiver/);
  assert.equal(channels.next(nilChannel), nilChannel, 'both nil-channel operations remain parked');

  const left = finish(channels, channels.createState('select-left'));
  const right = finish(channels, channels.createState('select-right'));
  assert.equal(left.view.selectedCase, 'left');
  assert.equal(right.view.selectedCase, 'right');
  assert.deepEqual(left.view.readyCasesAtSelection, ['left', 'right']);
  assert.deepEqual(right.view.readyCasesAtSelection, ['left', 'right']);
  assert.deepEqual(left.view.channels.find(({ id }) => id === 'left').buffer, []);
  assert.deepEqual(right.view.channels.find(({ id }) => id === 'right').buffer, []);
  assert.match(left.assumptions, /не гарант/i);
  assert.match(right.assumptions, /не гарант/i);
});

test('worker traces keep queued, running, completed, and cancelled work within the pool bound', () => {
  assert.deepEqual(workers.scenarios.map(({ id }) => id), ['normal-completion', 'cooperative-cancel']);

  for (const scenario of workers.scenarios) {
    let state = workers.createState(scenario.id);
    let guard = 20;
    while (true) {
      const statuses = state.view.jobs.map(({ status }) => status);
      const queuedIds = state.view.jobs.filter(({ status }) => status === 'queued').map(({ id }) => id);
      const runningJobIds = state.view.jobs.filter(({ status }) => status === 'running').map(({ id }) => id).sort();
      const assignedJobIds = state.view.workers.filter(({ status }) => status === 'running').map(({ jobId }) => jobId).sort();
      assert.deepEqual(state.view.queue, queuedIds);
      assert.deepEqual(assignedJobIds, runningJobIds, 'each running job belongs to exactly one worker');
      assert.equal(new Set(state.view.jobs.map(({ id }) => id)).size, state.view.jobs.length);
      assert.ok(state.view.workers.filter(({ status }) => status === 'running').length <= state.view.maxWorkers);
      for (const worker of state.view.workers.filter(({ status }) => status === 'running')) {
        assert.equal(state.view.jobs.find(({ id }) => id === worker.jobId).status, 'running');
      }
      if (state.view.joined) assert.equal(state.view.workers.filter(({ status }) => status === 'running').length, 0);
      assert.ok(statuses.every((status) => ['queued', 'running', 'completed', 'cancelled'].includes(status)));
      if (state.finished) break;
      assert.ok(guard > 0, 'worker trace should terminate');
      guard -= 1;
      state = workers.next(state);
    }
    assert.equal(state.view.joined, true);
    assert.equal(state.view.jobs.some(({ status }) => status === 'queued' || status === 'running'), false);
  }

  const normal = finish(workers, workers.createState('normal-completion'));
  assert.deepEqual(normal.view.result, [1, 2, 3, 4]);

  const cancelled = finish(workers, advance(workers, workers.createState('cooperative-cancel'), 2));
  assert.equal(cancelled.view.contextCancelled, true);
  assert.equal(cancelled.view.result, null, 'cancelled work does not publish a partial result');
  assert.deepEqual(cancelled.view.jobs.map(({ status }) => status), ['cancelled', 'cancelled', 'cancelled', 'cancelled']);
  assert.match(cancelled.assumptions, /кооператив/i);
  assert.match(cancelled.assumptions, /ожидает|жд[её]т/i);
});

test('model modules expose independent resettable immutable state', () => {
  for (const model of [slices, interfaces, defer, channels, workers]) {
    assert.ok(Array.isArray(model.scenarios) && model.scenarios.length > 0);
    for (const scenario of model.scenarios) {
      const first = model.createState(scenario.id);
      const independent = model.createState(scenario.id);
      const before = copy(first);
      const second = model.next(first);
      assert.deepEqual(first, before);
      assert.deepEqual(independent, before);
      assert.equal(first.goVersion, 'Go 1.27.1');
      assert.ok(first.assumptions.trim().length > 0);
      assert.ok(first.explanation.trim().length > 0);
      assert.ok(first.code.trim().length > 0);
      const reset = model.reset(second);
      assert.equal(reset.scenarioId, scenario.id);
      assert.equal(reset.stepIndex, 0);
      assert.equal(independent.stepIndex, 0, 'one scenario instance must not advance another');
    }
  }

  const firstInstance = slices.createState('spare-capacity');
  const secondInstance = slices.createState('spare-capacity');
  const advancedFirst = slices.next(firstInstance);
  assert.equal(secondInstance.stepIndex, 0);
  assert.notEqual(advancedFirst.view, secondInstance.view);
});


test('every model announces only its current step and explanation without moving focus', () => {
  const components = [
    'SlicesModel.vue',
    'InterfaceModel.vue',
    'DeferModel.vue',
    'ChannelsModel.vue',
    'WorkerPoolModel.vue',
  ];
  const statusElement = '<p class="model-live-status" role="status" aria-live="polite" aria-atomic="true">{{ statusAnnouncement }}</p>';
  const announcementExpression = "return this.announcementAction + ' Шаг ' + (this.state.stepIndex + 1) + ': ' + this.state.explanation;";
  for (const name of components) {
    const source = readFileSync(new URL('../components/' + name, import.meta.url), 'utf8');
    assert.ok(source.includes(statusElement), name + ' has a concise polite live status');
    assert.ok(source.includes(announcementExpression), name + ' announces step number and explanation only');
    assert.ok(source.includes('@click="step"'), name + ' retains the native step control');
    assert.ok(source.includes('@click="reset"'), name + ' retains the native reset control');
    assert.ok(source.includes('@change="selectScenario"'), name + ' retains the native scenario selector');
    assert.ok(source.includes("this.announcementAction = 'Выбран сценарий ' + this.currentScenario.label + '.';"), name + ' announces scenario changes');
    assert.ok(source.includes("this.announcementAction = 'Показан следующий шаг.';"), name + ' announces step changes');
    assert.ok(source.includes("this.announcementAction = 'Сценарий сброшен.';"), name + ' announces reset even at the initial step');
    assert.ok(!source.includes('.focus()') && !source.includes('nextTick('), name + ' does not move focus after a transition');
  }
});

test('slice SVG separates cell indices, capacity captions, and the next array row', () => {
  const source = readFileSync(new URL('../components/SlicesModel.vue', import.meta.url), 'utf8');
  const indexOffset = Number(source.match(/class="svg-index"[^>]*y="(\d+)"/)[1]);
  const rowPitch = Number(source.match(/\(14 \+ arrayIndex \* (\d+)\)/)[1]);
  const capacityOffset = Number(source.match(/class="svg-caption"[^>]*:y="(\d+) \+ arrayIndex \* \d+"/)[1]);
  const captionFontSize = Number(source.match(/\.svg-caption, \.svg-index \{[^}]*font-size: (\d+)px/)[1]);
  const indexBaseline = 14 + indexOffset;

  assert.ok(capacityOffset - indexBaseline >= captionFontSize * 1.5,
    'capacity caption must have a separate baseline below the index labels');
  assert.ok(14 + rowPitch - capacityOffset >= captionFontSize * 1.5,
    'the next array row must start below the capacity caption');
  assert.match(source, /18 \+ this\.state\.view\.backingArrays\.length \* \d+ \+ index \* 26/,
    'slice references must remain below all backing-array rows');
});

test('slice references route through left gutters and beneath only the target row', () => {
  const source = readFileSync(new URL('../components/SlicesModel.vue', import.meta.url), 'utf8');
  const method = source.match(/referencePath\(slice, index\) \{([\s\S]*?)\n    \},/)[1];

  assert.match(method, /const gutterX = 8 - index \* 4;/);
  assert.match(method, /const connectorY = this\.referenceY\(index\) \+ 8;/);
  assert.match(method, /const targetY = rowY \+ 36;/);
  assert.match(method, /const targetLaneY = targetY \+ 3;/);
  assert.match(method, /' L ' \+ gutterX \+ ' ' \+ targetLaneY \+ ' L ' \+ targetX \+ ' ' \+ targetLaneY \+ ' L ' \+ targetX \+ ' ' \+ targetY/,
    'the route rises in its gutter, then runs beneath the target cells to the target');
  assert.doesNotMatch(method, / C /, 'a straight gutter route cannot curve through an intermediate array heading');

  const gutters = [0, 1].map((sliceIndex) => 8 - sliceIndex * 4);
  assert.ok(gutters.every((gutterX) => gutterX < 12), 'each reference gutter stays left of array labels at x=12');
  assert.ok(gutters[0] > gutters[1], 'the upper source uses the inner gutter so its connector clears the lower route');
  for (const arrayIndex of [0, 1]) {
    const cellTop = 14 + arrayIndex * 112;
    const targetLane = cellTop + 39;
    const indexBaseline = cellTop + 54;
    assert.ok(targetLane > cellTop + 36, 'the route passes below cell rectangles');
    assert.ok(indexBaseline - targetLane >= 15, 'the route remains above index labels');
  }
});
