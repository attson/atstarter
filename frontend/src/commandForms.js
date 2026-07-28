import { envMapToText } from './envVars.js'

function lineFor(command) {
  return [command.command, ...(command.args || [])].filter(Boolean).join(' ')
}

export function commandFormsForProject(project) {
  if (!project) return []
  const source = project.commands && project.commands.length ? project.commands : [{
    id: 'default',
    name: 'Default',
    command: project.command,
    args: project.args || [],
    cwd: project.cwd || '',
    env: project.env || {},
    isDefault: true,
  }]
  return source.map((command, index) => ({
    id: command.id || '',
    name: command.name || (index === 0 ? 'Default' : `Command ${index + 1}`),
    line: lineFor(command),
    cwd: command.cwd || project.path || '',
    envText: envMapToText(command.env || {}),
    isDefault: !!command.isDefault || index === 0,
  }))
}
