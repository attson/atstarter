import * as vue from 'vue'
import { babelParse, compileScript, compileTemplate, parse } from 'vue/compiler-sfc'
import { transformSync } from 'esbuild'

export function compileComponent(source, dependencies) {
  const { descriptor } = parse(source)
  const script = compileScript(descriptor, { id: 'branch-ui-test' })
  const template = compileTemplate({
    source: descriptor.template.content,
    filename: 'BranchUiTest.vue',
    id: 'branch-ui-test',
    compilerOptions: { bindingMetadata: script.bindings, expressionPlugins: ['typescript'] },
  })
  let code = script.content.replace('export default ', 'const component = ') + '\n' +
    template.code.replace('export function render', 'function render')
  code = transformSync(code, { loader: 'ts', target: 'esnext' }).code
  const bindings = {}
  const imports = babelParse(code, { sourceType: 'module' }).program.body
    .filter((node) => node.type === 'ImportDeclaration')
  for (const node of imports.reverse()) {
    for (const specifier of node.specifiers) {
      const dependency = dependencies[node.source.value]
      bindings[specifier.local.name] = specifier.type === 'ImportDefaultSpecifier'
        ? dependency : dependency[specifier.imported.name]
    }
    code = code.slice(0, node.start) + code.slice(node.end)
  }
  return new Function(...Object.keys(bindings), code + '\ncomponent.render = render; return component')(...Object.values(bindings))
}


export function mountVueComponent(component, { props, i18n, beforeMount = () => {} }) {
  const node = (type, text = '') => Object.assign(new EventTarget(), {
    type, text, props: {}, style: {}, children: [], parent: null, focus() {}, select() {},
    getRootNode: () => globalThis.document,
  })
  const detach = (child) => {
    if (child.parent) child.parent.children.splice(child.parent.children.indexOf(child), 1)
    child.parent = null
  }
  const renderer = vue.createRenderer({
    createElement: (type) => node(type), createComment: (text) => node('#comment', text), createText: (text) => node('#text', text),
    setText: (element, text) => { element.text = text },
    setElementText: (element, text) => { element.text = text; element.children = [] },
    patchProp: (element, key, previous, next) => { element.props[key] = next },
    insert: (child, parent, anchor) => {
      detach(child)
      const index = anchor ? parent.children.indexOf(anchor) : -1
      parent.children.splice(index < 0 ? parent.children.length : index, 0, child)
      child.parent = parent
    },
    remove: detach,
    parentNode: (element) => element.parent,
    nextSibling: (element) => element.parent?.children[element.parent.children.indexOf(element) + 1] || null,
  })
  const reactiveProps = vue.reactive(props)
  const app = renderer.createApp({ setup: () => () => vue.h(component, reactiveProps) })
  app.use(i18n)
  const container = node('root')
  const originalDocument = globalThis.document
  const originalDocumentClass = globalThis.Document
  globalThis.Document = class extends EventTarget {}
  globalThis.document = new Document()
  beforeMount({ document: globalThis.document, app })
  const instance = app.mount(container)
  const all = (element = container) => [element, ...element.children.flatMap((child) => all(child))]
  const text = (element) => element.type === '#comment' ? '' : element.text + element.children.map(text).join('')
  return {
    state: instance.$.subTree.component.setupState,
    props: reactiveProps,
    find: (predicate) => all().find(predicate),
    visibleText: () => text(container),
    unmount: () => {
      try { app.unmount() } finally {
        if (originalDocument === undefined) delete globalThis.document
        else globalThis.document = originalDocument
        if (originalDocumentClass === undefined) delete globalThis.Document
        else globalThis.Document = originalDocumentClass
      }
    },
  }
}
