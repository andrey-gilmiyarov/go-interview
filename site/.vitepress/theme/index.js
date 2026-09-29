import { h } from 'vue'
import { useData } from 'vitepress'
import DefaultTheme from 'vitepress/theme-without-fonts'
import TopicMeta from './TopicMeta.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  Layout() {
    const { frontmatter } = useData()
    return h(DefaultTheme.Layout, null, {
      'doc-before': () => {
        const metadata = frontmatter.value
        if (!metadata.topicSummary || !metadata.goVersion) return null
        return h(TopicMeta, {
          summary: metadata.topicSummary,
          goVersion: metadata.goVersion,
        })
      },
    })
  },
}
