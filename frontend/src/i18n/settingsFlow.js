import { normalizeAppSettings } from '../appSettings.js'

export function applyLoadedSettings(raw, setPreference) {
  const settings = normalizeAppSettings(raw)
  setPreference(settings.language)
  return settings
}

export async function saveSettingsAndApply(raw, save, setPreference) {
  const settings = normalizeAppSettings(await save(raw))
  setPreference(settings.language)
  return settings
}
