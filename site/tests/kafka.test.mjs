import assert from 'node:assert/strict'
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { afterEach, test } from 'node:test'
import * as contentApi from '../scripts/content.mjs'

const temporaryRoots = []
const groupIds = ['language', 'api', 'concurrency', 'runtime', 'stdlib', 'testing', 'patterns', 'tooling']
const requiredHeadings = [
  'Коротко',
  'Как работает',
  'Пример',
  'Ошибки и ограничения',
  'Самопроверка',
  'Практика',
  'Источники',
]

function topic(overrides = {}) {
  return {
    id: 'topics-partitions',
    title: 'Топики и партиции',
    order: 1,
    summary: 'Модель журнала, ключи партиционирования и хранение событий.',
    kafkaVersion: '4.3.1',
    comparisonVersion: '3.9.2',
    reviewedAt: '2026-09-29',
    sources: ['https://kafka.apache.org/43/design/design/'],
    ...overrides,
  }
}

function makeRoot({ kafkaRecords, createPages = false } = {}) {
  const root = mkdtempSync(path.join(os.tmpdir(), 'go-handbook-kafka-'))
  temporaryRoots.push(root)
  mkdirSync(path.join(root, 'site/data/topics'), { recursive: true })
  writeFileSync(path.join(root, 'site/data/groups.json'), JSON.stringify(groupIds.map((id) => ({ id, title: id }))))
  if (kafkaRecords !== undefined) writeFileSync(path.join(root, 'site/data/kafka.json'), JSON.stringify(kafkaRecords, null, 2))
  if (createPages && Array.isArray(kafkaRecords)) {
    for (const record of kafkaRecords) {
      if (typeof record?.id !== 'string') continue
      writeKafkaPage(root, record.id)
    }
  }
  return root
}

function writeKafkaPage(root, id, headings = requiredHeadings) {
  mkdirSync(path.join(root, 'site/kafka'), { recursive: true })
  writeFileSync(path.join(root, 'site/kafka', id + '.md'), headings.map((heading) => '## ' + heading).join('\n\n'))
}

afterEach(() => {
  for (const root of temporaryRoots.splice(0)) rmSync(root, { recursive: true, force: true })
})

test('Kafka loader reads the reviewed registry and leaves the Go registry shape unchanged', () => {
  assert.equal(typeof contentApi.loadKafkaRegistry, 'function', 'content.mjs must export loadKafkaRegistry')
  const kafka = contentApi.loadKafkaRegistry()
  assert.deepEqual(kafka.map(({ id, title }) => [id, title]), [
    ['topics-partitions', 'Топики, партиции и журнал'],
    ['replication', 'Репликация и запись'],
    ['consumer-groups', 'Consumer groups'],
    ['offsets-delivery', 'Offsets и повторная доставка'],
    ['transactions', 'Идемпотентность и транзакции Kafka'],
    ['deduplication', 'Дедупликация и внешние эффекты'],
    ['outbox', 'Transactional outbox'],
    ['failures', 'Обработка сбоев'],
  ])
  assert.ok(kafka.every((record) => record.kafkaVersion === '4.3.1' && record.comparisonVersion === '3.9.2'))

  const root = makeRoot()
  const goRegistry = contentApi.loadRegistry({ root })
  assert.deepEqual(Object.keys(goRegistry), ['groups', 'topics', 'exercises'])
})

test('Kafka registry tolerates an absent file for partial work and reports it in full checks', () => {
  const root = makeRoot()
  assert.deepEqual(contentApi.loadKafkaRegistry({ root }), [])
  const partial = contentApi.checkContent({ root, partial: true })
  assert.deepEqual(partial.errors, [])
  assert.equal(partial.stats.kafkaTopics, 0)

  const full = contentApi.checkContent({ root })
  assert.match(full.errors.join('\n'), /Missing site\/data\/kafka\.json/)
  assert.match(full.errors.join('\n'), /Expected 8 Kafka topics, found 0/)
})

test('Kafka loader and checker diagnose malformed JSON and non-array registries once', () => {
  const root = makeRoot()
  mkdirSync(path.join(root, 'site/data'), { recursive: true })
  const registryPath = path.join(root, 'site/data/kafka.json')
  writeFileSync(registryPath, '{broken')
  assert.throws(() => contentApi.loadKafkaRegistry({ root }), /invalid JSON/)
  let errors = contentApi.checkContent({ root, partial: true }).errors
  assert.equal(errors.filter((error) => /kafka\.json: invalid JSON/.test(error)).length, 1)

  writeFileSync(registryPath, JSON.stringify({ id: 'not-an-array' }))
  assert.throws(() => contentApi.loadKafkaRegistry({ root }), /must contain an array/)
  errors = contentApi.checkContent({ root, partial: true }).errors
  assert.equal(errors.filter((error) => /kafka\.json must contain an array/.test(error)).length, 1)
})

test('Kafka sidebar includes overview and orders chapter links by registry order', () => {
  const root = makeRoot({ kafkaRecords: [
    topic({ id: 'second', title: 'Вторая глава', order: 2 }),
    topic({ id: 'first', title: 'Первая глава', order: 1 }),
  ] })
  assert.deepEqual(contentApi.buildKafkaSidebar({ root }), {
    text: 'Kafka',
    collapsed: false,
    items: [
      { text: 'Обзор', link: '/kafka/' },
      { text: 'Первая глава', link: '/kafka/first' },
      { text: 'Вторая глава', link: '/kafka/second' },
    ],
  })
})

test('Kafka checker accepts complete metadata and its required article headings', () => {
  const root = makeRoot({ kafkaRecords: [topic()], createPages: true })
  const report = contentApi.checkContent({ root, partial: true })
  assert.deepEqual(report.errors, [])
  assert.equal(report.stats.kafkaTopics, 1)
})

test('full Kafka content check accepts exactly eight topics when their pages exist', () => {
  const records = Array.from({ length: 8 }, (_, index) => topic({
    id: 'chapter-' + (index + 1),
    title: 'Глава ' + (index + 1),
    order: index + 1,
  }))
  const root = makeRoot({ kafkaRecords: records, createPages: true })
  const errors = contentApi.checkContent({ root }).errors.join('\n')
  assert.doesNotMatch(errors, /Expected 8 Kafka topics/)
})

test('Kafka checker reports empty required metadata fields', () => {
  const root = makeRoot({ kafkaRecords: [topic({
    title: ' ',
    summary: '',
    kafkaVersion: '',
    comparisonVersion: '',
    reviewedAt: '',
  })] })
  const errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  for (const field of ['title', 'summary', 'kafkaVersion', 'comparisonVersion', 'reviewedAt']) {
    assert.match(errors, new RegExp(field + ' must be a non-empty string'))
  }
})

test('Kafka checker rejects duplicate IDs and orders, reserved index IDs, and malformed records safely', () => {
  const root = makeRoot({ kafkaRecords: [
    topic(),
    topic({ id: 'topics-partitions', title: 'Дубликат', order: 1 }),
    topic({ id: 'index', order: 3 }),
    null,
  ] })
  const errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  assert.match(errors, /duplicate Kafka topic id "topics-partitions"/)
  assert.match(errors, /duplicate Kafka topic order 1/)
  assert.match(errors, /reserved.*index|index.*reserved/i)
  assert.match(errors, /Kafka topic record must be an object/)
})

test('Kafka checker validates semver versions and real ISO review dates', () => {
  const root = makeRoot({ kafkaRecords: [topic({
    kafkaVersion: '4.03.1',
    comparisonVersion: '3.9',
    reviewedAt: '2026-02-30',
  })] })
  const errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  assert.match(errors, /kafkaVersion must use x\.y\.z semantic version format/)
  assert.match(errors, /comparisonVersion must use x\.y\.z semantic version format/)
  assert.match(errors, /reviewedAt must be a real ISO date/)
})

test('Kafka checker applies source URL safeguards without echoing secrets', () => {
  const root = makeRoot({ kafkaRecords: [topic({
    sources: ['http://reader:secret@example.test/?token=secret'],
  })] })
  const errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  assert.match(errors, /source URL must use HTTPS/)
  assert.match(errors, /source URL contains credentials/)
  assert.match(errors, /authentication parameter "token"/)
  assert.doesNotMatch(errors, /reader|secret|example\.test|token=secret/)
})

test('Kafka checker reports missing pages and ignores required headings inside code fences', () => {
  const root = makeRoot({ kafkaRecords: [topic()] })
  let errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  assert.match(errors, /missing article site\/kafka\/topics-partitions\.md/)

  writeKafkaPage(root, 'topics-partitions', requiredHeadings.filter((heading) => heading !== 'Практика'))
  const pagePath = path.join(root, 'site/kafka/topics-partitions.md')
  writeFileSync(pagePath, readFileSync(pagePath, 'utf8') + '\n\n~~~md\n## Практика\n~~~')
  errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  assert.match(errors, /site\/kafka\/topics-partitions\.md: missing required heading "Практика"/)
})

test('Kafka content links under labs/kafka are traversed', () => {
  const root = makeRoot()
  mkdirSync(path.join(root, 'labs/kafka'), { recursive: true })
  writeFileSync(path.join(root, 'labs/kafka/README.md'), '[missing guide](./not-created.md)')
  const errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  assert.match(errors, /labs\/kafka\/README\.md: unresolved relative link path "\.\/not-created\.md"/)
})

test('local links to existing directories require an index.md or README.md landing page', () => {
  const root = makeRoot()
  mkdirSync(path.join(root, 'site/kafka'), { recursive: true })
  mkdirSync(path.join(root, 'site/empty-directory'), { recursive: true })
  mkdirSync(path.join(root, 'site/indexed-directory'), { recursive: true })
  mkdirSync(path.join(root, 'site/readme-directory'), { recursive: true })
  writeFileSync(path.join(root, 'site/indexed-directory/index.md'), 'Landing page')
  writeFileSync(path.join(root, 'site/readme-directory/README.md'), 'Landing page')
  writeFileSync(
    path.join(root, 'site/kafka/links.md'),
    '[empty](../empty-directory/)\n[index](../indexed-directory/)\n[readme](../readme-directory/)\n',
  )

  const errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  assert.match(errors, /site\/kafka\/links\.md: unresolved relative link path "\.\.\/empty-directory\/"/)
  assert.doesNotMatch(errors, /indexed-directory|readme-directory/)
})

test('malformed percent-encoded local links produce a generic diagnostic', () => {
  const root = makeRoot()
  mkdirSync(path.join(root, 'site/kafka'), { recursive: true })
  writeFileSync(path.join(root, 'site/kafka/links.md'), '[bad](../private-name%ZZ.md)')

  const errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')
  assert.match(errors, /site\/kafka\/links\.md: invalid percent-encoding in relative link path/)
  assert.doesNotMatch(errors, /private-name|%ZZ/)
})

test('Kafka nested spoilers and practice solution routes stay out of search text', () => {
  const html = '<p>видимый текст</p><details><summary>вопрос</summary><p>скрытый ответ</p><details><summary>вложено</summary>еще ответ</details></details>'
  const visible = contentApi.stripNonSearchableContent(html, '/kafka/offsets-delivery')
  assert.match(visible, /видимый текст/)
  assert.doesNotMatch(visible, /скрытый ответ|еще ответ/)
  assert.equal(contentApi.stripNonSearchableContent(html, '/kafka/practice/solutions/offset-replay'), '')
})
