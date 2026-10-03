import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitepress'
import { buildPostgresSidebar, loadPostgresRegistry, buildKafkaSidebar, buildSidebar, loadKafkaRegistry, loadRegistry, stripNonSearchableContent } from '../scripts/content.mjs'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const { topics, exercises } = loadRegistry({ root })
const postgresTopics = loadPostgresRegistry({ root })
const kafkaTopics = loadKafkaRegistry({ root })
const topicsBySourcePath = new Map(topics.map((topic) => [`go/${topic.group}/${topic.id}.md`, topic]))
const kafkaTopicsBySourcePath = new Map(kafkaTopics.map((topic) => [`kafka/${topic.id}.md`, topic]))

export default defineConfig({
  lang: 'ru-RU',
  title: 'Go и backend',
  description: 'Локальный справочник по Go, backend-разработке, Kafka и PostgreSQL.',
  cleanUrls: true,
  lastUpdated: false,
  markdown: {
    lineNumbers: true,
    include: { silent: false },
    snippet: { silent: false },
  },
  themeConfig: {
    siteTitle: 'Go и backend',
    nav: [
      { text: 'Главная', link: '/' },
      { text: 'Go', link: '/go/' },
      { text: 'Kafka', link: '/kafka/' },
      { text: 'PostgreSQL', link: '/postgres/' },
      { text: 'Практика', link: '/practice/' },
      { text: 'Источники', link: '/sources/' },
      { text: 'Дорожная карта', link: '/roadmap' },
    ],
    sidebar: {
      '/go/': buildSidebar({ root }),
      '/postgres/': [buildPostgresSidebar({ root }), { text: 'Лаборатория и задачи', items: [{ text: 'Запуск лаборатории', link: '/postgres/lab' }, { text: 'Интервью-практика', link: '/postgres/practice/' }] }],
      '/kafka/': [
        buildKafkaSidebar({ root }),
        {
          text: 'Лаборатория',
          collapsed: false,
          items: [
            { text: 'Запуск и сценарии', link: '/kafka/lab' },
            { text: 'CDC: Debezium и Kafka Connect', link: '/kafka/cdc-lab' },
          ],
        },
      ],
      '/practice/': [
        {
          text: 'Разбор поведения',
          collapsed: false,
          items: exercises.filter((exercise) => exercise.kind === 'trace').map((exercise) => ({
            text: exercise.title,
            link: '/practice/' + exercise.id,
          })),
        },
        {
          text: 'Реализация',
          collapsed: false,
          items: exercises.filter((exercise) => exercise.kind === 'implement').map((exercise) => ({
            text: exercise.title,
            link: '/practice/' + exercise.id,
          })),
        },
      ],
      '/solutions/': [],
      '/sources/': [{
        text: 'Материалы',
        collapsed: false,
        items: [
          { text: 'Каталог', link: '/sources/' },
          { text: '100 практических ошибок', link: '/sources/100-go-mistakes' },
          { text: '66 тем сообщества', link: '/sources/50-shades' },
        ],
      }],
    },
    outline: { level: [2, 3], label: 'На этой странице' },
    search: {
      provider: 'local',
      options: {
        async _render(source, env, markdown) {
          const html = markdown.render(source, env)
          if (env.frontmatter?.search === false) return ''
          return stripNonSearchableContent(html, env.relativePath ?? '')
        },
        locales: {
          root: {
            translations: {
              button: {
                buttonText: 'Поиск',
                buttonAriaLabel: 'Поиск по справочнику',
              },
              modal: {
                displayDetails: 'Показать подробности',
                resetButtonTitle: 'Сбросить поиск',
                backButtonTitle: 'Закрыть поиск',
                noResultsText: 'Ничего не найдено',
                footer: {
                  selectText: 'Выбрать',
                  selectKeyAriaLabel: 'Ввод',
                  navigateText: 'Перейти',
                  navigateUpKeyAriaLabel: 'Стрелка вверх',
                  navigateDownKeyAriaLabel: 'Стрелка вниз',
                  closeText: 'Закрыть',
                  closeKeyAriaLabel: 'Escape',
                },
              },
            },
          },
        },
      },
    },
    footer: { message: 'Справочник для локальной работы.' },
  },
  transformPageData(pageData) {
    const sourcePath = pageData.relativePath.replaceAll('\\', '/')
    const topic = topicsBySourcePath.get(sourcePath)
    if (topic) {
      pageData.frontmatter.topicSummary = topic.summary
      pageData.frontmatter.goVersion = topic.goVersion
    }
    const postgresTopic = postgresTopics.find((topic) => sourcePath === `postgres/${topic.id}.md`)
    if (postgresTopic) {
      pageData.frontmatter.topicSummary = postgresTopic.summary
      pageData.frontmatter.postgresVersion = postgresTopic.postgresVersion
      pageData.frontmatter.reviewedAt = postgresTopic.reviewedAt
    }
    const kafkaTopic = kafkaTopicsBySourcePath.get(sourcePath)
    if (kafkaTopic) {
      pageData.frontmatter.topicSummary = kafkaTopic.summary
      pageData.frontmatter.kafkaVersion = kafkaTopic.kafkaVersion
      pageData.frontmatter.comparisonVersion = kafkaTopic.comparisonVersion
      pageData.frontmatter.reviewedAt = kafkaTopic.reviewedAt
    }
  },
})
