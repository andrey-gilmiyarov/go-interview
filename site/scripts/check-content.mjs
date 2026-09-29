#!/usr/bin/env node

import { checkContent } from './content.mjs'

const args = new Set(process.argv.slice(2))
const unknown = [...args].filter((argument) => argument !== '--partial')
if (unknown.length) {
  console.error(`Unknown option${unknown.length === 1 ? '' : 's'}: ${unknown.join(', ')}`)
  process.exitCode = 2
} else {
  const report = checkContent({ partial: args.has('--partial') })
  if (report.errors.length) {
    console.error(report.errors.map((error) => `✗ ${error}`).join('\n'))
    console.error(`Checked ${report.stats.topics} Go topics, ${report.stats.exercises} exercises, and ${report.stats.kafkaTopics} Kafka topics.`)
    process.exitCode = 1
  } else {
    console.log(`Content check passed: ${report.stats.groups} groups, ${report.stats.topics} Go topics, ${report.stats.exercises} exercises, and ${report.stats.kafkaTopics} Kafka topics.`)
  }
}
