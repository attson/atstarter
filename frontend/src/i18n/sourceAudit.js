// Use the parsers shipped with the existing Vue dependency, rather than trying to
// split SFCs or strip comments with regular expressions. ASTs omit comments from
// the scan while retaining the original offsets for actionable diagnostics.
import { babelParse, parse as parseSfc } from 'vue/compiler-sfc'

export const AUDITED_UI_FILES = [
  'App.vue',
  'components/ConfirmDialog.vue',
  'components/DetectionSwitch.vue',
  'components/SettingsDialog.vue',
  'components/AddProjectDialog.vue',
  'components/EditProjectDialog.vue',
  'components/ProjectList.vue',
  'components/ProjectDetail.vue',
  'components/ProjectTreeNode.vue',
  'components/GroupDialog.vue',
  'components/GroupDetail.vue',
  'components/GroupTreeItem.vue',
  'components/GroupTreeNode.vue',
  'components/AddToGroupDialog.vue',
  'components/CommandRow.vue',
  'components/BranchSwitcher.vue',
  'components/ScanDialog.vue',
  'components/ui/AppButton.vue',
  'components/ui/AppIcon.vue',
  'components/ui/AppPill.vue',
  'components/ui/ThemeToggle.vue',
  'components/ui/ContextMenu.vue',
  'i18n/shellPresentation.js',
]

const accessibleAttributes = new Set(['title', 'aria-label', 'alt', 'placeholder'])
const presentationFields = new Set(['label', 'text', 'title', 'message', 'description', 'confirmText'])
const comparisons = new Set(['==', '===', '!=', '!==', '<', '>', '<=', '>='])
const eventCalls = new Set(['emit', '$emit', 'EventsOn', 'EventsOff', 'EventsEmit', 'addEventListener', 'removeEventListener'])
const han = /\p{Script=Han}/u

function propertyName(node) {
  if (!node) return undefined
  if (node.type === 'Identifier') return node.name
  if (node.type === 'StringLiteral') return node.value
  if (node.type === 'MemberExpression' || node.type === 'OptionalMemberExpression') {
    return node.computed ? node.property.value : node.property.name
  }
}

function isPresentationTarget(node) {
  if (presentationFields.has(propertyName(node))) return true
  return node?.type === 'MemberExpression' && propertyName(node) === 'value' && isPresentationTarget(node.object)
}

function isStaticString(node) {
  return node.type === 'StringLiteral' || (node.type === 'TemplateLiteral' && node.expressions.length === 0)
}

/**
 * Return source-ordered { line, text, kind } findings. `filename` identifies SFCs
 * and excludes catalogs; the caller adds it to diagnostics and applies exact
 * allowlist entries. Invalid input throws with the filename rather than passing.
 */
export function findHardcodedPresentationCopy(source, filename) {
  const normalizedFilename = filename.replaceAll('\\', '/')
  if (/(^|\/)i18n\/messages\//.test(normalizedFilename)) return []
  const findings = []
  function add(offset, text, kind) {
    const trimmed = text.trim()
    if (!trimmed) return
    findings.push({ offset, line: source.slice(0, offset).split('\n').length, text: trimmed, kind })
  }

  function scanScript(code, offset = 0, contextKind = null, expression = false) {
    const input = expression ? `(${code})` : code
    const baseOffset = offset - (expression ? 1 : 0)
    const ast = babelParse(input, { sourceType: 'module', plugins: ['typescript'], attachComment: false })

    function walk(node, kind = contextKind) {
      if (!node || typeof node !== 'object') return
      if (Array.isArray(node)) {
        node.forEach((child) => walk(child, kind))
        return
      }
      if (!node.type || node.type.endsWith('Comment')) return
      if (node.type.startsWith('Import') || node.type === 'ExportAllDeclaration') return
      if (node.type === 'ExportNamedDeclaration') {
        walk(node.declaration, kind)
        return
      }
      if (node.type === 'BinaryExpression' && comparisons.has(node.operator)) {
        // Direct strings are domain codes; nested calls/objects can own UI copy.
        for (const operand of [node.left, node.right]) {
          if (!isStaticString(operand)) walk(operand, kind)
        }
        return
      }
      if (node.type === 'MemberExpression' || node.type === 'OptionalMemberExpression') {
        walk(node.object, kind)
        // Lookup keys describe structure, including when forwarding err['message'].
        if (node.computed && !isStaticString(node.property)) walk(node.property, null)
        return
      }
      if (node.type === 'CallExpression' || node.type === 'OptionalCallExpression') {
        const name = propertyName(node.callee)
        if (name === 't' || name === '$t' || name === 'defineEmits' || name === 'require' || node.callee.type === 'Import') return
        walk(node.callee, kind)
        const args = eventCalls.has(name) ? node.arguments.slice(1) : node.arguments
        walk(args, kind)
        return
      }
      if (node.type === 'ObjectProperty') {
        const key = propertyName(node.key)
        if (key === 'data-testid' || key === 'testId' || key === 'testid') return
        walk(node.value, presentationFields.has(key) ? 'presentation-field' : kind)
        return
      }
      if (node.type === 'VariableDeclarator') {
        walk(node.init, isPresentationTarget(node.id) ? 'presentation-field' : kind)
        return
      }
      if (node.type === 'AssignmentExpression') {
        walk(node.right, isPresentationTarget(node.left) ? 'presentation-field' : kind)
        return
      }
      if (node.type === 'StringLiteral') {
        if (kind || han.test(node.value)) add(baseOffset + node.start, node.value, kind || 'script-string')
        return
      }
      if (node.type === 'TemplateLiteral') {
        const literalText = node.quasis.map((part) => part.value.cooked ?? part.value.raw).join('')
        // A dynamic value surrounded only by punctuation is formatting, e.g.
        // ` (${shell})`; there are no words to translate.
        if (han.test(literalText) || (kind && /\p{L}/u.test(literalText))) {
          add(baseOffset + node.start, input.slice(node.start + 1, node.end - 1), kind || 'script-string')
        }
        walk(node.expressions, kind)
        return
      }
      for (const [key, child] of Object.entries(node)) {
        if (!['loc', 'extra', 'comments', 'tokens'].includes(key)) walk(child, kind)
      }
    }
    walk(ast)
  }

  function scanTemplate(node, adjustment = 0) {
    if (node.type === 2) {
      const leadingWhitespace = node.loc.source.length - node.loc.source.trimStart().length
      add(node.loc.start.offset + adjustment + leadingWhitespace, node.content, 'template-text')
    } else if (node.type === 5) {
      scanScript(node.content.content, node.content.loc.start.offset + adjustment, 'template-expression', true)
    } else if (node.type === 1) {
      if (node.tag === 'script' || node.tag === 'style') return
      for (const prop of node.props) {
        if (prop.type === 6 && accessibleAttributes.has(prop.name) && prop.value) {
          add(prop.value.loc.start.offset + adjustment, prop.value.content, 'attribute')
        } else if (prop.type === 7 && prop.name === 'bind' && prop.arg?.isStatic && accessibleAttributes.has(prop.arg.content) && prop.exp) {
          scanScript(prop.exp.content, prop.exp.loc.start.offset + adjustment, 'attribute', true)
        }
      }
    }
    node.children?.forEach((child) => scanTemplate(child, adjustment))
  }

  try {
    if (normalizedFilename.endsWith('.vue')) {
      const { descriptor, errors } = parseSfc(source, { filename })
      if (errors.length) throw errors[0]
      if (descriptor.template) {
        if (descriptor.template.lang && descriptor.template.lang !== 'html') throw new Error('Only HTML templates are supported')
        scanTemplate(descriptor.template.ast)
      } else if (descriptor.customBlocks.length) {
        // Also support small fixture/template fragments alongside a script block.
        const prefix = '<template>'
        const fragment = parseSfc(`${prefix}${source}</template>`, { filename })
        if (fragment.errors.length) throw fragment.errors[0]
        scanTemplate(fragment.descriptor.template.ast, -prefix.length)
      }
      for (const block of [descriptor.script, descriptor.scriptSetup]) {
        if (block?.src) throw new Error('External scripts must be audited as separate files')
        if (block) scanScript(block.content, block.loc.start.offset)
      }
    } else {
      scanScript(source)
    }
  } catch (error) {
    throw new Error(`${filename}: source audit could not parse input: ${error.message || error}`, { cause: error })
  }
  return findings.sort((a, b) => a.offset - b.offset).map(({ line, text, kind }) => ({ line, text, kind }))
}
