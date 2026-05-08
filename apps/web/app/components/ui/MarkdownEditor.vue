<template>
  <div
    class="border border-neutral-200 dark:border-neutral-700 rounded-lg overflow-hidden focus-within:ring-2 focus-within:ring-primary-500 focus-within:border-transparent transition-shadow"
  >
    <!-- Tab bar -->
    <div class="flex border-b border-neutral-200 dark:border-neutral-700 bg-neutral-50 dark:bg-neutral-900/50">
      <button
        v-for="t in tabs"
        :key="t.id"
        type="button"
        class="px-4 py-1.5 text-xs font-medium transition-colors border-b-2 -mb-px"
        :class="tab === t.id
          ? 'border-primary-500 text-neutral-900 dark:text-white bg-white dark:bg-neutral-900'
          : 'border-transparent text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-300'"
        @click="switchTab(t.id)"
      >{{ t.label }}</button>
    </div>

    <!-- Write: raw markdown textarea -->
    <textarea
      v-show="tab === 'write'"
      :value="modelValue"
      :rows="rows"
      :placeholder="placeholder"
      :required="required && tab === 'write'"
      class="w-full bg-white dark:bg-neutral-900 px-3 py-2.5 text-sm text-neutral-900 dark:text-white placeholder-neutral-400 focus:outline-none resize-y font-mono leading-relaxed"
      @input="onTextareaInput"
    />

    <!-- Visual: Tiptap WYSIWYG -->
    <div v-show="tab === 'visual'" class="bg-white dark:bg-neutral-900">
      <div v-if="editor" class="flex items-center gap-0.5 px-2 py-1 border-b border-neutral-100 dark:border-neutral-800 flex-wrap">
        <!-- Bold -->
        <button type="button" title="Bold" class="md-tbtn" :class="{ 'is-active': editor.isActive('bold') }" @click="editor.chain().focus().toggleBold().run()">
          <strong class="text-xs font-bold leading-none">B</strong>
        </button>
        <!-- Italic -->
        <button type="button" title="Italic" class="md-tbtn" :class="{ 'is-active': editor.isActive('italic') }" @click="editor.chain().focus().toggleItalic().run()">
          <em class="text-xs italic font-semibold leading-none">I</em>
        </button>
        <!-- Strike -->
        <button type="button" title="Strikethrough" class="md-tbtn" :class="{ 'is-active': editor.isActive('strike') }" @click="editor.chain().focus().toggleStrike().run()">
          <s class="text-xs font-medium leading-none">S</s>
        </button>

        <div class="w-px h-4 bg-neutral-200 dark:bg-neutral-700 mx-1" />

        <!-- Bullet list -->
        <button type="button" title="Bullet list" class="md-tbtn" :class="{ 'is-active': editor.isActive('bulletList') }" @click="editor.chain().focus().toggleBulletList().run()">
          <svg class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="currentColor">
            <circle cx="2" cy="4" r="1.25"/>
            <circle cx="2" cy="8" r="1.25"/>
            <circle cx="2" cy="12" r="1.25"/>
            <rect x="5" y="3.25" width="9" height="1.5" rx="0.75"/>
            <rect x="5" y="7.25" width="9" height="1.5" rx="0.75"/>
            <rect x="5" y="11.25" width="9" height="1.5" rx="0.75"/>
          </svg>
        </button>
        <!-- Ordered list -->
        <button type="button" title="Numbered list" class="md-tbtn" :class="{ 'is-active': editor.isActive('orderedList') }" @click="editor.chain().focus().toggleOrderedList().run()">
          <svg class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="currentColor">
            <text x="0.5" y="5.5" font-size="5" font-family="monospace">1.</text>
            <text x="0.5" y="9.5" font-size="5" font-family="monospace">2.</text>
            <text x="0.5" y="13.5" font-size="5" font-family="monospace">3.</text>
            <rect x="6" y="3.25" width="8" height="1.5" rx="0.75"/>
            <rect x="6" y="7.25" width="8" height="1.5" rx="0.75"/>
            <rect x="6" y="11.25" width="8" height="1.5" rx="0.75"/>
          </svg>
        </button>
        <!-- Blockquote -->
        <button type="button" title="Blockquote" class="md-tbtn" :class="{ 'is-active': editor.isActive('blockquote') }" @click="editor.chain().focus().toggleBlockquote().run()">
          <svg class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
            <path d="M2 12V6a2 2 0 0 1 2-2h1"/>
            <path d="M9 12V6a2 2 0 0 1 2-2h1"/>
          </svg>
        </button>

        <div class="w-px h-4 bg-neutral-200 dark:bg-neutral-700 mx-1" />

        <!-- Undo -->
        <button type="button" title="Undo" class="md-tbtn" :disabled="!editor.can().undo()" @click="editor.chain().focus().undo().run()">
          <svg class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 6h6a4 4 0 0 1 0 8H5"/>
            <path d="M2 3l3 3-3 3"/>
          </svg>
        </button>
        <!-- Redo -->
        <button type="button" title="Redo" class="md-tbtn" :disabled="!editor.can().redo()" @click="editor.chain().focus().redo().run()">
          <svg class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 6H8a4 4 0 0 0 0 8h3"/>
            <path d="M14 3l-3 3 3 3"/>
          </svg>
        </button>
      </div>

      <!-- Tiptap content area -->
      <EditorContent
        v-if="mounted"
        :editor="editor"
        class="md-visual-editor px-3 py-2.5 min-h-30"
      />
      <div v-else class="px-3 py-2.5 min-h-30" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from 'tiptap-markdown'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  rows?: number
  required?: boolean
}>(), {
  placeholder: 'Write something…',
  rows: 6,
  required: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

type Tab = 'write' | 'visual'
const tab = ref<Tab>('write')
const mounted = ref(false)

const tabs: { id: Tab; label: string }[] = [
  { id: 'write', label: 'Write' },
  { id: 'visual', label: 'Visual' },
]

const editor = useEditor({
  extensions: [
    StarterKit,
    Markdown.configure({ html: false, transformCopiedText: true }),
  ],
  content: props.modelValue || '',
  editorProps: {
    attributes: { class: 'outline-none' },
  },
  onUpdate({ editor }) {
    emit('update:modelValue', editor.storage.markdown.getMarkdown())
  },
})

onMounted(() => { mounted.value = true })
onBeforeUnmount(() => { editor.value?.destroy() })

function switchTab(newTab: Tab) {
  if (newTab === tab.value) return
  if (newTab === 'visual' && editor.value) {
    editor.value.commands.setContent(props.modelValue || '')
  }
  tab.value = newTab
}

function onTextareaInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLTextAreaElement).value)
}
</script>

<style scoped>
@reference "~/assets/css/main.css";

.md-tbtn {
  @apply p-1.5 rounded text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer;
}
.md-tbtn.is-active {
  @apply bg-neutral-200 dark:bg-neutral-700 text-neutral-900 dark:text-white;
}
</style>

<style>
/* ProseMirror content styles — unscoped, namespaced by .md-visual-editor */
.md-visual-editor .ProseMirror {
  outline: none;
  font-size: 0.875rem;
  line-height: 1.6;
  color: inherit;
}
.md-visual-editor .ProseMirror > * + * {
  margin-top: 0.4em;
}
.md-visual-editor .ProseMirror p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  color: #9ca3af;
  pointer-events: none;
  float: left;
  height: 0;
}
.md-visual-editor .ProseMirror strong { font-weight: 600; }
.md-visual-editor .ProseMirror em { font-style: italic; }
.md-visual-editor .ProseMirror s { text-decoration: line-through; }
.md-visual-editor .ProseMirror ul {
  list-style-type: disc;
  padding-left: 1.4em;
}
.md-visual-editor .ProseMirror ol {
  list-style-type: decimal;
  padding-left: 1.4em;
}
.md-visual-editor .ProseMirror li + li { margin-top: 0.15em; }
.md-visual-editor .ProseMirror blockquote {
  border-left: 3px solid #d1d5db;
  padding-left: 0.75em;
  color: #6b7280;
  font-style: italic;
  margin: 0.25em 0;
}
.dark .md-visual-editor .ProseMirror blockquote {
  border-left-color: #4b5563;
  color: #9ca3af;
}
.md-visual-editor .ProseMirror code {
  background: #f3f4f6;
  border-radius: 3px;
  padding: 0.1em 0.3em;
  font-size: 0.85em;
  font-family: ui-monospace, monospace;
}
.dark .md-visual-editor .ProseMirror code {
  background: #1f2937;
}
.md-visual-editor .ProseMirror pre {
  background: #f3f4f6;
  border-radius: 6px;
  padding: 0.75em 1em;
  overflow-x: auto;
}
.dark .md-visual-editor .ProseMirror pre {
  background: #111827;
}
.md-visual-editor .ProseMirror pre code {
  background: transparent;
  padding: 0;
  font-size: 0.8rem;
}
.md-visual-editor .ProseMirror h1 { font-size: 1.25em; font-weight: 700; }
.md-visual-editor .ProseMirror h2 { font-size: 1.1em; font-weight: 700; }
.md-visual-editor .ProseMirror h3 { font-size: 1em; font-weight: 600; }
</style>
