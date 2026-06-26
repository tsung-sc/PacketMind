import { onMounted, onUnmounted, ref, type Ref } from 'vue'

export function useNarrowContainer(containerRef: Ref<HTMLElement | null>, breakpoint = 400) {
  const isNarrow = ref(false)
  let observer: ResizeObserver | null = null

  const update = (width: number) => {
    isNarrow.value = width < breakpoint
  }

  onMounted(() => {
    const element = containerRef.value
    if (!element) {
      return
    }

    update(element.clientWidth)

    observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        update(entry.contentRect.width)
      }
    })

    observer.observe(element)
  })

  onUnmounted(() => {
    observer?.disconnect()
    observer = null
  })

  return {
    isNarrow,
  }
}
