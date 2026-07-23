import { typeLabel } from './typeLabel.js'

export function detectionOptionsFor(project) {
  if (!project) return []
  const options = []
  const seen = new Set()
  for (const option of project.detectionOptions || []) {
    if (!option || !option.type || seen.has(option.type)) continue
    seen.add(option.type)
    options.push({
      type: option.type,
      command: option.command || '',
      args: option.args || [],
    })
  }
  if (project.detectedType && !seen.has(project.detectedType)) {
    options.unshift({
      type: project.detectedType,
      command: project.command || '',
      args: project.args || [],
    })
  }
  return options
}

export function hasDetectionSwitch(project) {
  return detectionOptionsFor(project).length > 1
}

export function canIgnoreComposeDetection(project) {
  return project?.detectedType === 'compose' &&
    detectionOptionsFor(project).some((option) => option.type !== 'compose')
}

export function detectionLabel(option) {
  return typeLabel(option && option.type)
}

export function commandLineForDetection(option) {
  if (!option) return ''
  return [option.command, ...(option.args || [])].filter(Boolean).join(' ')
}

export function applyDetectionOption(project, option) {
  const command = option.command || ''
  const args = option.args || []
  const next = {
    ...project,
    detectedType: option.type,
    command,
    args,
    autoDetected: false,
  }
  if (option.type === 'compose' || !command) {
    next.commands = []
    return next
  }
  next.commands = [{
    id: 'default',
    name: 'Default',
    command,
    args,
    cwd: project.cwd || '',
    env: project.env || {},
    isDefault: true,
  }]
  return next
}

export function ignoreComposeDetections(projects) {
  return (projects || []).map((project) => {
    if (!canIgnoreComposeDetection(project)) return project
    const fallback = detectionOptionsFor(project).find((option) => option.type !== 'compose')
    return fallback ? applyDetectionOption(project, fallback) : project
  })
}
