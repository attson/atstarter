#!/usr/bin/env node
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = process.cwd()
const pluginName = 'atstarter-control'
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
assert.equal(codexMarketplace.name, 'atstarter')
assert.equal(codexMarketplace.interface?.displayName, 'atstarter')
const codexEntry = codexMarketplace.plugins?.find((plugin) => plugin.name === pluginName)
assert.ok(codexEntry, 'Codex marketplace must include atstarter-control')
assert.deepEqual(codexEntry.source, {
  source: 'local',
  path: './plugins/atstarter-control',
})
assert.equal(codexEntry.policy?.installation, 'AVAILABLE')
assert.equal(codexEntry.policy?.authentication, 'ON_INSTALL')
assert.equal(codexEntry.category, 'Developer Tools')

const claudeMarketplacePath = join(root, '.claude-plugin', 'marketplace.json')
const claudeMarketplace = readJSON(claudeMarketplacePath)
assert.equal(claudeMarketplace.name, 'atstarter')
assert.equal(claudeMarketplace.owner?.name, 'atstarter')
const claudeEntry = claudeMarketplace.plugins?.find((plugin) => plugin.name === pluginName)
assert.ok(claudeEntry, 'Claude marketplace must include atstarter-control')
assert.equal(claudeEntry.source, './plugins/atstarter-control')

const codexPlugin = readJSON(join(pluginPath, '.codex-plugin', 'plugin.json'))
assert.equal(codexPlugin.name, pluginName)
assert.equal(codexPlugin.skills, './skills/')
assert.equal(codexPlugin.mcpServers, './.mcp.json')

const claudePlugin = readJSON(join(pluginPath, '.claude-plugin', 'plugin.json'))
assert.equal(claudePlugin.name, pluginName)
assert.equal(claudePlugin.skills?.[0], './skills/use-atstarter')
assert.equal(claudePlugin.mcpServers?.atstarter?.command, 'atstarter')
assert.deepEqual(claudePlugin.mcpServers.atstarter.args, ['mcp'])

const mcpConfig = readJSON(join(pluginPath, '.mcp.json'))
assert.equal(mcpConfig.mcpServers?.atstarter?.command, 'atstarter')
assert.deepEqual(mcpConfig.mcpServers.atstarter.args, ['mcp'])
assertFile(join(pluginPath, 'skills', 'use-atstarter', 'SKILL.md'))
