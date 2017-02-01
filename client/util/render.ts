// Utility functions for reducing layout thrashing, by batching DOM writes and
// reads. Basically a stripped down version of FastDOM.
// Also contains utilities for HTML template tags.

import { followDOM } from "./scroll"

// Cached element containing the banners, posts, catalog and other board HTML
export const threads = document.getElementById("threads")

// Holds cached references to all out HTML template tags' contents
const templates: { [name: string]: DocumentFragment } = {}

let readStack: Operation[] = [],
    writeStack: Operation[] = [],
    scheduled: boolean

type Operation = () => void

// Import a prepared template and return it's HTML contents
export function importTemplate(name: string): DocumentFragment {
    return document.importNode(templates[name], true) as DocumentFragment
}

// Schedule a DOM write operation
export function write(operation: Operation) {
    writeStack.push(operation)
    scheduleFlush()
}

// Schedule a DOM read operation
export function read(operation: Operation) {
    readStack.push(operation)
    scheduleFlush()
}

// Schedule a flush on the next animation frame, if not yet scheduled
function scheduleFlush() {
    if (!scheduled) {
        scheduled = true
        requestAnimationFrame(() =>
            followDOM(flush))
    }
}

// Perform all write tasks and then read tasks in the stack
function flush() {
    const writes = writeStack,
        reads = readStack
    writeStack = []
    readStack = []
    scheduled = false
    for (let i = 0; i < writes.length; i++) {
        writes[i]()
    }
    for (let i = 0; i < reads.length; i++) {
        reads[i]()
    }
}

// Load HTML templates
for (let el of document.head.querySelectorAll("template")) {
    templates[el.getAttribute("name")] = (el as HTMLTemplateElement).content
}

// Toggle an optional style element in the head
export function toggleHeadStyle(
    name: string,
    css: string,
): (toggle: boolean) => void {
    return toggle => {
        const id = name + "-toggle"
        if (!document.getElementById(id)) {
            const html = `<style id="${id}">${css}</style>`
            write(() =>
                document.head.append(makeEl(html)))
        }

        // The disabled property only exists on elements in the DOM, so we do
        // another query
        write(() =>
            (document.getElementById(id) as HTMLInputElement)
                .disabled = !toggle)
    }
}

// Parse HTML string to a single Node
export function makeEl(DOMString: string): Node {
    const el = document.createElement('div')
    el.innerHTML = DOMString
    return el.firstChild
}