import { detectionLabel } from '../projectDetection.js'

// Keep presentation out of confirmation state so an open dialog follows locale
// changes and continues to describe its current domain payload.
export function confirmationPresentation({ kind, payload }, t) {
  if (kind === 'remove') return {
    title: t('dialogs.removeContainer.title'),
    message: t(payload?.state === 'running'
      ? 'dialogs.removeContainer.runningMessage'
      : 'dialogs.removeContainer.message', { name: payload?.name || '' }),
    confirmText: t('common.remove'),
  }
  if (kind === 'down') return {
    title: t('dialogs.composeDown.title'),
    message: t('dialogs.composeDown.message'),
    confirmText: t('dialogs.composeDown.confirm'),
  }
  return {
    title: t('dialogs.resetProjects.title'),
    message: t('dialogs.resetProjects.message'),
    confirmText: t('common.reset'),
  }
}

export function summaryTotalLabel(activeTab, projectCount, containerCount, t) {
  return activeTab === 'containers'
    ? t('navigation.containerCount', containerCount, { count: containerCount })
    : t('navigation.projectCount', projectCount, { count: projectCount })
}

export function detectionChoiceLabel(option, t) {
  return !option?.type || option.type === 'unknown' ? t('status.unknown') : detectionLabel(option)
}
