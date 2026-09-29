import assert from 'node:assert/strict'
import { existsSync, mkdtempSync, mkdirSync, readFileSync, readdirSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { afterEach, test } from 'node:test'
import * as contentApi from '../scripts/content.mjs'

const temporaryRoots = []

function makeRoot() {
  const root = mkdtempSync(path.join(os.tmpdir(), 'go-handbook-content-'))
  temporaryRoots.push(root)
  mkdirSync(path.join(root, 'site/data/topics'), { recursive: true })
  mkdirSync(path.join(root, 'site/go/language'), { recursive: true })
  writeFileSync(path.join(root, 'site/data/groups.json'), JSON.stringify([
    { id: 'language', title: 'Язык Go' },
    { id: 'api', title: 'API и контракты' },
    { id: 'concurrency', title: 'Конкурентность' },
    { id: 'runtime', title: 'Runtime и память' },
    { id: 'stdlib', title: 'Стандартная библиотека' },
    { id: 'testing', title: 'Тестирование' },
    { id: 'patterns', title: 'Паттерны' },
    { id: 'tooling', title: 'Инструменты' },
  ], null, 2))
  writeFileSync(path.join(root, 'site/data/topics/language.json'), JSON.stringify([
    {
      id: 'slices',
      title: 'Срезы',
      group: 'language',
      order: 1,
      summary: 'Заголовок, массив и емкость среза.',
      goVersion: '1.27.1',
      sources: ['https://go.dev/ref/spec#Slice_types'],
      exercises: [],
    },
  ], null, 2))
  writeFileSync(path.join(root, 'site/go/language/slices.md'), [
    '# Срезы',
    '## Коротко',
    '## Как работает',
    '## Пример',
    '## Ошибки и ограничения',
    '## Самопроверка',
    '<details><summary>Ответы</summary>Содержимое подсказки.</details>',
    '## Практика',
    '## Источники',
  ].join('\n\n'))
  return root
}

function collectFiles(directory, files = []) {
  if (!existsSync(directory)) return files
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const fullPath = path.join(directory, entry.name)
    if (entry.isDirectory()) collectFiles(fullPath, files)
    else if (entry.isFile()) files.push(fullPath)
  }
  return files
}

afterEach(() => {
  for (const root of temporaryRoots.splice(0)) rmSync(root, { recursive: true, force: true })
})

test('content checker accepts a complete partial registry and its article', () => {
  assert.equal(typeof contentApi.checkContent, 'function', 'site/scripts/content.mjs must export checkContent')
  const report = contentApi.checkContent({ root: makeRoot(), partial: true })
  assert.deepEqual(report.errors, [])
  assert.equal(report.stats.topics, 1)
})

test('registry loader tolerates topic files that are not written yet', () => {
  const registry = contentApi.loadRegistry({ root: makeRoot() })
  assert.equal(registry.groups.length, 8)
  assert.deepEqual(registry.topics.map((topic) => topic.id), ['slices'])
})

test('content checker rejects duplicate slugs and metadata whose group disagrees with its file', () => {
  assert.equal(typeof contentApi.checkContent, 'function', 'site/scripts/content.mjs must export checkContent')
  const root = makeRoot()
  const metadataPath = path.join(root, 'site/data/topics/language.json')
  const metadata = JSON.parse(readFileSync(metadataPath, 'utf8'))
  metadata.push({ ...metadata[0], group: 'api' })
  writeFileSync(metadataPath, JSON.stringify(metadata, null, 2))
  const report = contentApi.checkContent({ root, partial: true })
  assert.match(report.errors.join('\n'), /duplicate topic id.*slices/i)
  assert.match(report.errors.join('\n'), /group.*language|language.*group/i)
})

test('content checker rejects source URLs with credentials or tracking parameters', () => {
  assert.equal(typeof contentApi.checkContent, 'function', 'site/scripts/content.mjs must export checkContent')
  const root = makeRoot()
  const metadataPath = path.join(root, 'site/data/topics/language.json')
  const metadata = JSON.parse(readFileSync(metadataPath, 'utf8'))
  metadata[0].sources = [
    'https://reader:private@go.dev/ref/spec?utm_source=mail',
    'https://go.dev/ref/spec#id_token=private-fragment',
  ]
  writeFileSync(metadataPath, JSON.stringify(metadata, null, 2))
  const report = contentApi.checkContent({ root, partial: true })
  assert.match(report.errors.join('\n'), /source.*credential|credential.*source/i)
  assert.match(report.errors.join('\n'), /tracking|utm_source/i)
  assert.match(report.errors.join('\n'), /id_token/i)
  assert.doesNotMatch(report.errors.join('\n'), /reader|private|mail/)
})

test('source URL validation never echoes credential-bearing non-HTTPS URLs', () => {
  const root = makeRoot()
  const metadataPath = path.join(root, 'site/data/topics/language.json')
  const metadata = JSON.parse(readFileSync(metadataPath, 'utf8'))
  metadata[0].sources = ['http://reader:secret@example.test/?token=secret']
  writeFileSync(metadataPath, JSON.stringify(metadata, null, 2))
  const errors = contentApi.checkContent({ root, partial: true }).errors.join('\n')

  assert.match(errors, /source URL must use HTTPS/i)
  assert.match(errors, /source URL contains credentials/i)
  assert.match(errors, /authentication parameter "token"/i)
  assert.doesNotMatch(errors, /reader|secret|example\.test|token=secret/)
})

test('built theme CSS uses system fonts without remote font assets', () => {
  const root = mkdtempSync(path.join(os.tmpdir(), 'go-handbook-theme-'))
  temporaryRoots.push(root)
  symlinkSync(path.join(contentApi.REPOSITORY_ROOT, 'node_modules'), path.join(root, 'node_modules'), 'dir')
  mkdirSync(path.join(root, '.vitepress'), { recursive: true })
  symlinkSync(path.join(contentApi.REPOSITORY_ROOT, 'site/.vitepress/theme'), path.join(root, '.vitepress/theme'), 'dir')
  writeFileSync(path.join(root, 'index.md'), '# CSS smoke test\n')
  writeFileSync(path.join(root, '.vitepress/config.mjs'), [
    "import { defineConfig } from 'vitepress'",
    "export default defineConfig({ title: 'CSS smoke test', themeConfig: { search: { provider: 'local' } } })",
  ].join('\n'))

  const vitepressPath = path.join(contentApi.REPOSITORY_ROOT, 'node_modules/vitepress/bin/vitepress.js')
  const result = spawnSync(process.execPath, [vitepressPath, 'build', root], {
    cwd: contentApi.REPOSITORY_ROOT,
    encoding: 'utf8',
  })
  assert.equal(result.status, 0, `${result.stdout}\n${result.stderr}`)
  const cssFiles = collectFiles(path.join(root, '.vitepress/dist')).filter((file) => file.endsWith('.css'))
  assert.ok(cssFiles.length > 0, 'VitePress should emit theme CSS')
  const css = cssFiles.map((file) => readFileSync(file, 'utf8')).join('\n')

  assert.match(css, /--vp-font-family-base\s*:\s*ui-sans-serif/i)
  assert.doesNotMatch(css, /@font-face/i)
  assert.doesNotMatch(css, /fonts\.(?:googleapis|gstatic)\.com/i)
  assert.doesNotMatch(css, /@import\s+url\(\s*['"]?https?:/i)
  assert.doesNotMatch(css, /url\(\s*['"]?https?:/i)
})

test('topic and exercise registries must reference each other', () => {
  const root = makeRoot()
  const metadataPath = path.join(root, 'site/data/topics/language.json')
  const metadata = JSON.parse(readFileSync(metadataPath, 'utf8'))
  metadata[0].exercises = ['slice-alias']
  writeFileSync(metadataPath, JSON.stringify(metadata, null, 2))
  mkdirSync(path.join(root, 'site/data'), { recursive: true })
  writeFileSync(path.join(root, 'site/data/exercises.json'), JSON.stringify([
    { id: 'slice-alias', title: 'Общий массив', kind: 'trace', topics: [] },
  ], null, 2))
  const report = contentApi.checkContent({ root, partial: true })
  assert.match(report.errors.join('\n'), /does not reference topic "slices"/)
})

test('required headings inside code examples do not satisfy the article structure', () => {
  assert.equal(typeof contentApi.checkContent, 'function', 'site/scripts/content.mjs must export checkContent')
  const root = makeRoot()
  const articlePath = path.join(root, 'site/go/language/slices.md')
  const markdown = readFileSync(articlePath, 'utf8').replace(
    '## Практика',
    '```md\n## Практика\n```',
  )
  writeFileSync(articlePath, markdown)
  const report = contentApi.checkContent({ root, partial: true })
  assert.match(report.errors.join('\n'), /missing required heading "Практика"/)
})

test('content checker reports broken relative links and unresolved VitePress includes', () => {
  assert.equal(typeof contentApi.checkContent, 'function', 'site/scripts/content.mjs must export checkContent')
  const root = makeRoot()
  writeFileSync(path.join(root, 'site/go/language/slices.md'), [
    '## Коротко',
    '[несуществующая страница](./missing.md)',
    '```md',
    '<<< @/../practice/not-yet-created/README.md',
    '```',
    '<!--@include: @/../practice/no-such-part/README.md-->',
    '## Как работает',
    '## Пример',
    '## Ошибки и ограничения',
    '## Самопроверка',
    '## Практика',
    '## Источники',
  ].join('\n'))
  const report = contentApi.checkContent({ root, partial: true })
  assert.match(report.errors.join('\n'), /missing\.md/)
  assert.match(report.errors.join('\n'), /not-yet-created\/README\.md/)
  assert.match(report.errors.join('\n'), /no-such-part\/README\.md/)
})

test('search extraction removes all native details bodies and solution routes', () => {
  assert.equal(typeof contentApi.stripNonSearchableContent, 'function', 'site/scripts/content.mjs must export stripNonSearchableContent')
  const html = '<p>видимый текст</p><details><summary>вопрос</summary><p>скрытый ответ</p><details><summary>вложено</summary>еще ответ</details></details><pre data-search-exclude><code>секретный код</code></pre>'
  const visible = contentApi.stripNonSearchableContent(html, '/go/language/slices')
  assert.match(visible, /видимый текст/)
  assert.doesNotMatch(visible, /скрытый ответ|еще ответ|секретный код/)
  assert.equal(contentApi.stripNonSearchableContent(html, '/solutions/slices'), '')
})

test('content checker validates repository README and docs links', () => {
  const root = makeRoot()
  mkdirSync(path.join(root, 'docs'), { recursive: true })
  writeFileSync(path.join(root, 'README.md'), '[guide](docs/missing.md)')
  writeFileSync(path.join(root, 'docs/authoring.md'), '[missing](./missing.md)')

  const report = contentApi.checkContent({ root, partial: true })
  assert.match(report.errors.join('\n'), /README\.md: unresolved relative link path/)
  assert.match(report.errors.join('\n'), /docs\/authoring\.md: unresolved relative link path/)
})

test('content checker rejects a missing canonical exercise file', () => {
  const root = makeRoot()
  writeFileSync(path.join(root, 'site/data/exercises.json'), JSON.stringify([
    { id: 'slice-alias', title: 'Общий массив', kind: 'trace', topics: [] },
  ], null, 2))

  const report = contentApi.checkContent({ root, partial: true })
  assert.match(report.errors.join('\n'), /practice\/slice-alias\/README\.md/)
})

test('strict content check requires the approved topic and exercise totals', () => {
  assert.equal(typeof contentApi.checkContent, 'function', 'site/scripts/content.mjs must export checkContent')
  const report = contentApi.checkContent({ root: makeRoot(), partial: false })
  assert.match(report.errors.join('\n'), /42 topics/)
  assert.match(report.errors.join('\n'), /20 exercises/)
})
