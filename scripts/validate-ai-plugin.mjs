#!/usr/bin/env node
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = process.cwd()
const pluginName = 'rootyard-control'
const pluginPath = join(root, 'plugins', pluginName)

function readJSON(path) {
  assert.ok(existsSync(path), `missing ${path}`)
  return JSON.parse(readFileSync(path, 'utf8'))
}

function assertFile(path) {
  assert.ok(existsSync(path), `missing ${path}`)
}

const codexMarketplacePath = join(root, '.agents', 'plugins', 'marketplace.json')
const codexMarketplace = readJSON(codexMarketplacePath)
assert.equal(codexMarketplace.name, 'rootyard')
assert.equal(codexMarketplace.interface?.displayName, 'Rootyard')
const codexEntry = codexMarketplace.plugins?.find((plugin) => plugin.name === pluginName)
assert.ok(codexEntry, 'Codex marketplace must include rootyard-control')
assert.deepEqual(codexEntry.source, {
  source: 'local',
  path: './plugins/rootyard-control',
})
assert.equal(codexEntry.policy?.installation, 'AVAILABLE')
assert.equal(codexEntry.policy?.authentication, 'ON_INSTALL')
assert.equal(codexEntry.category, 'Developer Tools')

const claudeMarketplacePath = join(root, '.claude-plugin', 'marketplace.json')
const claudeMarketplace = readJSON(claudeMarketplacePath)
assert.equal(claudeMarketplace.name, 'rootyard')
assert.equal(claudeMarketplace.owner?.name, 'Rootyard')
const claudeEntry = claudeMarketplace.plugins?.find((plugin) => plugin.name === pluginName)
assert.ok(claudeEntry, 'Claude marketplace must include rootyard-control')
assert.equal(claudeEntry.source, './plugins/rootyard-control')

const codexPlugin = readJSON(join(pluginPath, '.codex-plugin', 'plugin.json'))
assert.equal(codexPlugin.name, pluginName)
assert.equal(codexPlugin.skills, './skills/')
assert.equal(codexPlugin.mcpServers, './.mcp.json')

const claudePlugin = readJSON(join(pluginPath, '.claude-plugin', 'plugin.json'))
assert.equal(claudePlugin.name, pluginName)
assert.equal(claudePlugin.skills?.[0], './skills/use-rootyard')
assert.equal(claudePlugin.mcpServers?.rootyard?.command, 'rootyard')
assert.deepEqual(claudePlugin.mcpServers.rootyard.args, ['mcp'])

const mcpConfig = readJSON(join(pluginPath, '.mcp.json'))
assert.equal(mcpConfig.mcpServers?.rootyard?.command, 'rootyard')
assert.deepEqual(mcpConfig.mcpServers.rootyard.args, ['mcp'])
assertFile(join(pluginPath, 'skills', 'use-rootyard', 'SKILL.md'))
