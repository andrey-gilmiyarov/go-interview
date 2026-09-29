import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export const GROUP_IDS = Object.freeze([
  'language',
  'api',
  'concurrency',
  'runtime',
  'stdlib',
  'testing',
  'patterns',
  'tooling',
])

export const REQUIRED_HEADINGS = Object.freeze([
  'Коротко',
  'Как работает',
  'Пример',
  'Ошибки и ограничения',
  'Самопроверка',
  'Практика',
  'Источники',
])

const REQUIRED_EXERCISE_FILES = Object.freeze([
  'README.md',
  'SOLUTION.md',
  'starter/starter.go',
  'starter/starter_test.go',
  'solution/solution.go',
  'solution/solution_test.go',
  'internal/checks/checks.go',
])

export const REPOSITORY_ROOT = fileURLToPath(new URL('../../', import.meta.url))

function readJSON(filePath, label, errors, fallback) {
  if (!existsSync(filePath)) return fallback
  try {
    return JSON.parse(readFileSync(filePath, 'utf8'))
  } catch (error) {
    errors.push(`${label}: invalid JSON (${error.message}).`)
    return fallback
  }
}

function walkMarkdown(directory, files = []) {
  if (!existsSync(directory)) return files
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (entry.name === '.vitepress' || entry.name === 'node_modules') continue
    const fullPath = path.join(directory, entry.name)
    if (entry.isDirectory()) walkMarkdown(fullPath, files)
    else if (entry.isFile() && entry.name.endsWith('.md')) files.push(fullPath)
  }
  return files
}

function readRegistry(root) {
  const errors = []
  const siteRoot = path.join(root, 'site')
  const groupsPath = path.join(siteRoot, 'data/groups.json')
  const groups = readJSON(groupsPath, 'site/data/groups.json', errors, [])
  if (!Array.isArray(groups)) errors.push('site/data/groups.json must contain an array.')

  const topics = []
  for (const groupId of GROUP_IDS) {
    const groupPath = path.join(siteRoot, `data/topics/${groupId}.json`)
    const records = readJSON(groupPath, `site/data/topics/${groupId}.json`, errors, [])
    if (!Array.isArray(records)) {
      errors.push(`site/data/topics/${groupId}.json must contain an array.`)
      continue
    }
    topics.push(...records)
  }

  const exercisesPath = path.join(siteRoot, 'data/exercises.json')
  const hasExerciseManifest = existsSync(exercisesPath)
  const exercises = readJSON(exercisesPath, 'site/data/exercises.json', errors, [])
  if (hasExerciseManifest && !Array.isArray(exercises)) {
    errors.push('site/data/exercises.json must contain an array.')
  }
  return { errors, groups: Array.isArray(groups) ? groups : [], topics, exercises: Array.isArray(exercises) ? exercises : [], hasExerciseManifest }
}

function readKafkaRegistry(root) {
  const errors = []
  const filePath = path.join(root, 'site/data/kafka.json')
  const exists = existsSync(filePath)
  const parsed = readJSON(filePath, 'site/data/kafka.json', errors, [])
  if (exists && !Array.isArray(parsed)) errors.push('site/data/kafka.json must contain an array.')
  return { errors, records: Array.isArray(parsed) ? parsed : [], exists }
}

/**
 * Load the metadata needed by the site. Topic files that have not been written
 * yet are deliberately treated as empty so independent authors can work in
 * parallel without making the development server unstartable.
 */
export function loadRegistry({ root = REPOSITORY_ROOT } = {}) {
  const registry = readRegistry(root)
  if (registry.errors.length) throw new Error(registry.errors.join('\n'))
  return { groups: registry.groups, topics: registry.topics, exercises: registry.exercises }
}

export function buildSidebar({ root = REPOSITORY_ROOT } = {}) {
  const { groups, topics } = loadRegistry({ root })
  return groups.map((group) => ({
    text: group.title,
    collapsed: false,
    items: topics
      .filter((topic) => topic.group === group.id)
      .sort((left, right) => left.order - right.order || left.title.localeCompare(right.title, 'ru'))
      .map((topic) => ({
        text: topic.title,
        link: `/go/${group.id}/${topic.id}`,
      })),
  }))
}

/** Load the Kafka topic metadata without changing the Go registry contract. */
export function loadKafkaRegistry({ root = REPOSITORY_ROOT } = {}) {
  const registry = readKafkaRegistry(root)
  if (registry.errors.length) throw new Error(registry.errors.join('\n'))
  return registry.records
}

/** Build the VitePress navigation group for the Kafka handbook. */
export function buildKafkaSidebar({ root = REPOSITORY_ROOT } = {}) {
  const records = loadKafkaRegistry({ root })
  const items = records
    .slice()
    .sort((left, right) => {
      const leftOrder = Number.isInteger(left?.order) && left.order > 0 ? left.order : Number.MAX_SAFE_INTEGER
      const rightOrder = Number.isInteger(right?.order) && right.order > 0 ? right.order : Number.MAX_SAFE_INTEGER
      return leftOrder - rightOrder || String(left?.title ?? '').localeCompare(String(right?.title ?? ''), 'ru')
    })
    .map((record) => ({
      text: typeof record?.title === 'string' ? record.title : '',
      link: '/kafka/' + (typeof record?.id === 'string' ? record.id : ''),
    }))
  return {
    text: 'Kafka',
    collapsed: false,
    items: [{ text: 'Обзор', link: '/kafka/' }, ...items],
  }
}

function isExternalTarget(target) {
  return /^(?:[a-z][a-z\d+.-]*:|\/\/)/i.test(target)
}

function isRegularFile(filePath) {
  try {
    return statSync(filePath).isFile()
  } catch {
    return false
  }
}

function isDirectory(directoryPath) {
  try {
    return statSync(directoryPath).isDirectory()
  } catch {
    return false
  }
}

function resolveLocalFile(root, fromFile, rawTarget) {
  let target = rawTarget.trim().replace(/^<|>$/g, '')
  if (!target || target.startsWith('#') || isExternalTarget(target)) return null
  target = target.split(/[?#]/, 1)[0]
  try {
    target = decodeURIComponent(target)
  } catch {
    return { invalidEncoding: true }
  }
  const siteRoot = path.join(root, 'site')
  let candidate
  if (target.startsWith('@/')) candidate = path.resolve(siteRoot, target.slice(2))
  else if (target.startsWith('/')) candidate = path.resolve(siteRoot, `.${target}`)
  else candidate = path.resolve(path.dirname(fromFile), target)

  const rootPrefix = `${path.resolve(root)}${path.sep}`
  if (candidate !== path.resolve(root) && !candidate.startsWith(rootPrefix)) return { path: candidate }
  if (isRegularFile(candidate)) return { path: candidate }
  if (isDirectory(candidate)) {
    const landingPage = ['index.md', 'README.md']
      .map((name) => path.join(candidate, name))
      .find(isRegularFile)
    return { path: landingPage ?? candidate }
  }

  const candidates = [candidate]
  if (!path.extname(candidate)) candidates.push(`${candidate}.md`, path.join(candidate, 'index.md'), path.join(candidate, 'README.md'))
  return { path: candidates.find(isRegularFile) ?? candidate }
}

function stripCodeFences(markdown) {
  const lines = markdown.split(/\r?\n/)
  const kept = []
  let fence = null
  for (const line of lines) {
    const match = line.match(/^\s*(`{3,}|~{3,})/)
    if (!fence && match) {
      fence = { marker: match[1][0], length: match[1].length }
      continue
    }
    if (fence) {
      const close = line.match(/^\s*(`{3,}|~{3,})\s*$/)
      if (close && close[1][0] === fence.marker && close[1].length >= fence.length) fence = null
      continue
    }
    kept.push(line)
  }
  return kept.join('\n').replace(/`[^`\n]*`/g, '')
}

function checkMarkdownReferences(root, filePath, errors) {
  const markdown = readFileSync(filePath, 'utf8')
  const source = stripCodeFences(markdown)
  const rel = path.relative(root, filePath).split(path.sep).join('/')
  const references = []
  const linkPattern = /!?\[[^\]]*\]\(\s*(?:<([^>]+)>|([^\s)]+))/g
  for (const match of source.matchAll(linkPattern)) references.push({ target: match[1] ?? match[2], kind: 'link' })
  const htmlPattern = /(?:href|src)\s*=\s*["']([^"']+)["']/gi
  for (const match of source.matchAll(htmlPattern)) references.push({ target: match[1], kind: 'link' })
  const snippetPattern = /^\s*<<<\s+([^\s]+)/gm
  for (const match of markdown.matchAll(snippetPattern)) {
    references.push({ target: match[1].replace(/\{.*$/, '').replace(/\[[^\]]*$/, ''), kind: 'include' })
  }
  const includePattern = /<!--\s*@include:\s*([^\s}]+)(?:\{[^}]*\})?\s*-->/g
  for (const match of markdown.matchAll(includePattern)) references.push({ target: match[1], kind: 'include' })

  for (const reference of references) {
    const resolution = resolveLocalFile(root, filePath, reference.target)
    if (!resolution) continue
    if (resolution.invalidEncoding) {
      errors.push(`${rel}: invalid percent-encoding in relative ${reference.kind} path.`)
      continue
    }
    const targetPath = resolution.path
    if (!isRegularFile(targetPath) || !path.resolve(targetPath).startsWith(`${path.resolve(root)}${path.sep}`)) {
      errors.push(`${rel}: unresolved relative ${reference.kind} path "${reference.target}".`)
    }
  }
}

function validateSourceURL(source, location, errors) {
  if (typeof source !== 'string' || !source.trim()) {
    errors.push(`${location}: source URLs must be non-empty strings.`)
    return
  }
  let parsed
  try {
    parsed = new URL(source)
  } catch {
    errors.push(`${location}: invalid source URL.`)
    return
  }
  if (parsed.protocol !== 'https:') errors.push(`${location}: source URL must use HTTPS.`)
  if (parsed.username || parsed.password) errors.push(`${location}: source URL contains credentials.`)
  const parameterSets = [parsed.searchParams, new URLSearchParams(parsed.hash.replace(/^#/, ''))]
  for (const parameters of parameterSets) {
    for (const key of parameters.keys()) {
      if (/^utm_/i.test(key) || /(?:token|secret|password|authorization|session|api[_-]?key|client[_-]?secret)/i.test(key)) {
        errors.push(`${location}: source URL contains tracking or authentication parameter "${key}".`)
      }
    }
  }
}

function topicHeadings(markdown) {
  return new Set([...markdown.matchAll(/^#{2,3}\s+(.+?)\s*#*\s*$/gm)].map((match) => match[1].trim()))
}

function validateMetadataTopic(topic, groupId, index, root, knownIds, errors) {
  const location = `site/data/topics/${groupId}.json[${index}]`
  if (!topic || typeof topic !== 'object' || Array.isArray(topic)) {
    errors.push(`${location}: topic record must be an object.`)
    return
  }
  for (const key of ['id', 'title', 'group', 'summary', 'goVersion']) {
    if (typeof topic[key] !== 'string' || !topic[key].trim()) errors.push(`${location}: ${key} must be a non-empty string.`)
  }
  if (typeof topic.id === 'string') {
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(topic.id)) errors.push(`${location}: id "${topic.id}" must be a kebab-case slug.`)
    if (knownIds.has(topic.id)) errors.push(`${location}: duplicate topic id "${topic.id}".`)
    knownIds.add(topic.id)
  }
  if (topic.group !== groupId) errors.push(`${location}: group must match registry file group "${groupId}".`)
  if (!Number.isInteger(topic.order) || topic.order < 1) errors.push(`${location}: order must be a positive integer.`)
  if (typeof topic.goVersion === 'string' && !/^\d+\.\d+\.\d+$/.test(topic.goVersion)) errors.push(`${location}: goVersion must use x.y.z format.`)
  if (!Array.isArray(topic.sources) || topic.sources.length === 0) errors.push(`${location}: sources must be a non-empty array.`)
  else topic.sources.forEach((source, sourceIndex) => validateSourceURL(source, `${location}.sources[${sourceIndex}]`, errors))
  if (!Array.isArray(topic.exercises) || topic.exercises.some((id) => typeof id !== 'string' || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(id))) {
    errors.push(`${location}: exercises must be an array of exercise IDs.`)
  }

  if (typeof topic.id !== 'string') return
  const articlePath = path.join(root, 'site', 'go', groupId, `${topic.id}.md`)
  if (!existsSync(articlePath)) {
    errors.push(`${location}: missing article site/go/${groupId}/${topic.id}.md.`)
    return
  }
  const markdown = readFileSync(articlePath, 'utf8')
  const headings = topicHeadings(stripCodeFences(markdown))
  for (const heading of REQUIRED_HEADINGS) {
    if (!headings.has(heading)) errors.push(`site/go/${groupId}/${topic.id}.md: missing required heading "${heading}".`)
  }
}

function isSemanticVersion(version) {
  return typeof version === 'string' && /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(version)
}

function isRealISODate(value) {
  if (typeof value !== 'string' || !/^(?!0000)\d{4}-\d{2}-\d{2}$/.test(value)) return false
  const date = new Date(value + 'T00:00:00.000Z')
  return Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === value
}

function validateKafkaTopic(topic, index, root, knownIds, knownOrders, errors) {
  const location = 'site/data/kafka.json[' + index + ']'
  if (!topic || typeof topic !== 'object' || Array.isArray(topic)) {
    errors.push(location + ': Kafka topic record must be an object.')
    return
  }

  for (const key of ['id', 'title', 'summary', 'kafkaVersion', 'comparisonVersion', 'reviewedAt']) {
    if (typeof topic[key] !== 'string' || !topic[key].trim()) {
      errors.push(location + ': ' + key + ' must be a non-empty string.')
    }
  }

  const validId = typeof topic.id === 'string' && /^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(topic.id)
  if (typeof topic.id === 'string') {
    if (!validId) errors.push(location + ': id "' + topic.id + '" must be a kebab-case slug.')
    else if (topic.id === 'index') errors.push(location + ': id "index" is reserved for the Kafka overview.')
    else if (knownIds.has(topic.id)) errors.push(location + ': duplicate Kafka topic id "' + topic.id + '".')
    else knownIds.add(topic.id)
  }

  if (!Number.isInteger(topic.order) || topic.order < 1) {
    errors.push(location + ': order must be a positive integer.')
  } else if (knownOrders.has(topic.order)) {
    errors.push(location + ': duplicate Kafka topic order ' + topic.order + '.')
  } else {
    knownOrders.add(topic.order)
  }

  if (typeof topic.kafkaVersion === 'string' && !isSemanticVersion(topic.kafkaVersion)) {
    errors.push(location + ': kafkaVersion must use x.y.z semantic version format.')
  }
  if (typeof topic.comparisonVersion === 'string' && !isSemanticVersion(topic.comparisonVersion)) {
    errors.push(location + ': comparisonVersion must use x.y.z semantic version format.')
  }
  if (typeof topic.reviewedAt === 'string' && !isRealISODate(topic.reviewedAt)) {
    errors.push(location + ': reviewedAt must be a real ISO date in YYYY-MM-DD format.')
  }
  if (!Array.isArray(topic.sources) || topic.sources.length === 0) {
    errors.push(location + ': sources must be a non-empty array.')
  } else {
    topic.sources.forEach((source, sourceIndex) => validateSourceURL(source, location + '.sources[' + sourceIndex + ']', errors))
  }

  if (!validId || topic.id === 'index') return
  const articlePath = path.join(root, 'site/kafka', topic.id + '.md')
  const articleName = 'site/kafka/' + topic.id + '.md'
  if (!existsSync(articlePath)) {
    errors.push(location + ': missing article ' + articleName + '.')
    return
  }
  const markdown = readFileSync(articlePath, 'utf8')
  const headings = topicHeadings(stripCodeFences(markdown))
  for (const heading of REQUIRED_HEADINGS) {
    if (!headings.has(heading)) errors.push(articleName + ': missing required heading "' + heading + '".')
  }
}

/** Validate the metadata registry and all local Markdown references. */
export function checkContent({ root = REPOSITORY_ROOT, partial = false } = {}) {
  const errors = []
  const registry = readRegistry(root)
  errors.push(...registry.errors)
  const groupIds = registry.groups.map((group) => group?.id)
  if (JSON.stringify(groupIds) !== JSON.stringify(GROUP_IDS)) {
    errors.push(`site/data/groups.json must list groups in this order: ${GROUP_IDS.join(', ')}.`)
  }
  for (const [index, group] of registry.groups.entries()) {
    if (!group || typeof group !== 'object' || typeof group.title !== 'string' || !group.title.trim()) {
      errors.push(`site/data/groups.json[${index}]: title must be a non-empty string.`)
    }
  }

  const knownTopicIds = new Set()
  for (const groupId of GROUP_IDS) {
    const topicPath = path.join(root, 'site', `data/topics/${groupId}.json`)
    if (!existsSync(topicPath)) {
      if (!partial) errors.push(`Missing site/data/topics/${groupId}.json.`)
      continue
    }
    const topics = readJSON(topicPath, `site/data/topics/${groupId}.json`, errors, [])
    if (!Array.isArray(topics)) {
      errors.push(`site/data/topics/${groupId}.json must contain an array.`)
      continue
    }
    topics.forEach((topic, index) => validateMetadataTopic(topic, groupId, index, root, knownTopicIds, errors))
  }

  const kafkaRegistry = readKafkaRegistry(root)
  errors.push(...kafkaRegistry.errors)
  if (!partial && !kafkaRegistry.exists) errors.push('Missing site/data/kafka.json.')
  const knownKafkaIds = new Set()
  const knownKafkaOrders = new Set()
  kafkaRegistry.records.forEach((topic, index) => {
    validateKafkaTopic(topic, index, root, knownKafkaIds, knownKafkaOrders, errors)
  })

  const exercisePath = path.join(root, 'site/data/exercises.json')
  if (!registry.hasExerciseManifest) {
    if (!partial) errors.push('Missing site/data/exercises.json.')
  } else {
    const exerciseIds = new Set()
    for (const [index, exercise] of registry.exercises.entries()) {
      const location = `site/data/exercises.json[${index}]`
      if (!exercise || typeof exercise !== 'object' || Array.isArray(exercise)) {
        errors.push(`${location}: exercise record must be an object.`)
        continue
      }
      if (typeof exercise.id !== 'string' || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(exercise.id)) {
        errors.push(`${location}: id must be a kebab-case slug.`)
      } else if (exerciseIds.has(exercise.id)) {
        errors.push(`${location}: duplicate exercise id "${exercise.id}".`)
      } else {
        exerciseIds.add(exercise.id)
        for (const relativePath of REQUIRED_EXERCISE_FILES) {
          const filePath = path.join(root, 'practice', exercise.id, relativePath)
          if (!existsSync(filePath)) {
            errors.push(`Exercise "${exercise.id}" is missing required file practice/${exercise.id}/${relativePath}.`)
          }
        }
      }
      if (typeof exercise.title !== 'string' || !exercise.title.trim()) errors.push(`${location}: title must be a non-empty string.`)
      if (!['trace', 'implement'].includes(exercise.kind)) errors.push(`${location}: kind must be trace or implement.`)
      if (!Array.isArray(exercise.topics) || exercise.topics.some((id) => typeof id !== 'string')) errors.push(`${location}: topics must be an array of topic IDs.`)
    }
    for (const topic of registry.topics) {
      if (!topic || !Array.isArray(topic.exercises)) continue
      for (const exerciseId of topic.exercises) {
        const exercise = registry.exercises.find((candidate) => candidate?.id === exerciseId)
        if (!exercise) errors.push(`Topic "${topic.id}" references unknown exercise "${exerciseId}".`)
        else if (!exercise.topics?.includes(topic.id)) errors.push(`Exercise "${exerciseId}" does not reference topic "${topic.id}".`)
      }
    }
    for (const exercise of registry.exercises) {
      if (!Array.isArray(exercise?.topics)) continue
      for (const topicId of exercise.topics) {
        const topic = registry.topics.find((candidate) => candidate?.id === topicId)
        if (!topic && !partial) errors.push(`Exercise "${exercise.id}" references unknown topic "${topicId}".`)
        else if (topic && !topic.exercises?.includes(exercise.id)) errors.push(`Topic "${topicId}" does not reference exercise "${exercise.id}".`)
      }
    }
  }

  const repositoryMarkdownFiles = [
    path.join(root, 'README.md'),
    ...walkMarkdown(path.join(root, 'docs')),
  ].filter((filePath) => existsSync(filePath))
  const markdownFiles = [
    ...walkMarkdown(path.join(root, 'site')),
    ...walkMarkdown(path.join(root, 'practice')),
    ...walkMarkdown(path.join(root, 'labs/kafka')),
    ...repositoryMarkdownFiles,
  ]
  for (const filePath of markdownFiles) checkMarkdownReferences(root, filePath, errors)

  if (!partial && registry.topics.length !== 42) errors.push(`Expected 42 topics, found ${registry.topics.length}.`)
  if (!partial && registry.exercises.length !== 20) errors.push(`Expected 20 exercises, found ${registry.exercises.length}.`)
  if (!partial && kafkaRegistry.records.length !== 8) {
    errors.push('Expected 8 Kafka topics, found ' + kafkaRegistry.records.length + '.')
  }
  return {
    errors,
    stats: {
      groups: registry.groups.length,
      topics: registry.topics.length,
      exercises: registry.exercises.length,
      kafkaTopics: kafkaRegistry.records.length,
    },
  }
}

function removeBalancedElement(html, tagName) {
  const tagPattern = new RegExp(`<\\/?${tagName}\\b[^>]*>`, 'gi')
  let output = ''
  let cursor = 0
  let depth = 0
  for (const match of html.matchAll(tagPattern)) {
    if (depth === 0) output += html.slice(cursor, match.index)
    const closing = /^<\//.test(match[0])
    if (closing) depth = Math.max(0, depth - 1)
    else depth += 1
    cursor = match.index + match[0].length
  }
  if (depth === 0) output += html.slice(cursor)
  return output
}

/** Remove spoiler bodies and solution pages before VitePress indexes text. */
export function stripNonSearchableContent(html, route = '') {
  if (/(?:^|\/)solutions(?:\/|$)/.test(route.replace(/^\/?site\//, '/'))) return ''
  return removeBalancedElement(String(html), 'details')
    .replace(/<pre\b[^>]*\bdata-search-exclude\b[^>]*>[\s\S]*?<\/pre>/gi, '')
}
