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

// 新增命令表单:cwd 直接落入 project.path 作为真实可编辑值(与既有命令的空 cwd
// 回填行为一致),而非仅作 placeholder,否则保存时 cwd 会是空串。
export function blankCommandForm(project, count) {
  return {
    id: '',
    name: `Command ${count + 1}`,
    line: '',
    cwd: (project && project.path) || '',
    envText: '',
    isDefault: count === 0,
  }
}
