// MIT-licensed excerpts from inertiajs/inertia v2.3.18.
// Commit ed9b159a5857663211580e081d6bd90520f17bad; see LICENSE and README.md.
// Only dependencies/browser events are doubled; the methods below are unchanged.
type Page = any
let current: Page
const currentPage = { get: () => current, hasOnceProps: () => false }
const get = (obj: any, path: string): any => path.split('.').reduce((o, key) => o?.[key], obj)
const set = (obj: any, path: string, value: any) => {
  const keys = path.split('.'); const last = keys.pop()!; let at = obj
  for (const key of keys) at = at[key] ??= {}
  at[last] = value
}
export function setCurrent(page: Page) { current = page }
export class ResponseReplay {
  requestParams = { isPartial: () => true, isDeferredPropsRequest: () => false }
  apply(page: Page) { this.mergeProps(page) }
  protected mergeProps(pageResponse: Page): void {
    if (!this.requestParams.isPartial() || pageResponse.component !== currentPage.get().component) {
      return
    }

    const propsToAppend = pageResponse.mergeProps || []
    const propsToPrepend = pageResponse.prependProps || []
    const propsToDeepMerge = pageResponse.deepMergeProps || []
    const matchPropsOn = pageResponse.matchPropsOn || []

    const mergeProp = (prop: string, shouldAppend: boolean) => {
      const currentProp = get(currentPage.get().props, prop)
      const incomingProp = get(pageResponse.props, prop)

      if (Array.isArray(incomingProp)) {
        const newArray = this.mergeOrMatchItems(
          (currentProp || []) as any[],
          incomingProp,
          prop,
          matchPropsOn,
          shouldAppend,
        )

        set(pageResponse.props, prop, newArray)
      } else if (typeof incomingProp === 'object' && incomingProp !== null) {
        const newObject = {
          ...(currentProp || {}),
          ...incomingProp,
        }

        set(pageResponse.props, prop, newObject)
      }
    }

    propsToAppend.forEach((prop) => mergeProp(prop, true))
    propsToPrepend.forEach((prop) => mergeProp(prop, false))

    propsToDeepMerge.forEach((prop) => {
      const currentProp = currentPage.get().props[prop]
      const incomingProp = pageResponse.props[prop]

      // Function to recursively merge objects and arrays
      const deepMerge = (target: any, source: any, matchProp: string) => {
        if (Array.isArray(source)) {
          return this.mergeOrMatchItems(target, source, matchProp, matchPropsOn)
        }

        if (typeof source === 'object' && source !== null) {
          // Merge objects by iterating over keys
          return Object.keys(source).reduce(
            (acc, key) => {
              acc[key] = deepMerge(target ? target[key] : undefined, source[key], `${matchProp}.${key}`)
              return acc
            },
            { ...target },
          )
        }

        // If the source is neither an array nor an object, simply return the it
        return source
      }

      // Apply the deep merge and update the page response
      pageResponse.props[prop] = deepMerge(currentProp, incomingProp, prop)
    })

    pageResponse.props = { ...currentPage.get().props, ...pageResponse.props }

    if (this.requestParams.isDeferredPropsRequest()) {
      const currentErrors = currentPage.get().props.errors

      if (currentErrors && Object.keys(currentErrors).length > 0) {
        // Preserve existing errors during deferred props requests
        pageResponse.props.errors = currentErrors
      }
    }

    // Preserve the existing scrollProps
    if (currentPage.get().scrollProps) {
      pageResponse.scrollProps = {
        ...(currentPage.get().scrollProps || {}),
        ...(pageResponse.scrollProps || {}),
      }
    }

    // Preserve the existing onceProps
    if (currentPage.hasOnceProps()) {
      pageResponse.onceProps = {
        ...(currentPage.get().onceProps || {}),
        ...(pageResponse.onceProps || {}),
      }
    }

    // Preserve flash data and merge with new flash data on non-deferred requests
    pageResponse.flash = {
      ...currentPage.get().flash,
      ...(this.requestParams.isDeferredPropsRequest() ? {} : pageResponse.flash),
    }

    const currentOriginalDeferred = currentPage.get().initialDeferredProps
    if (currentOriginalDeferred && Object.keys(currentOriginalDeferred).length > 0) {
      pageResponse.initialDeferredProps = currentOriginalDeferred
    }
  }

  protected mergeOrMatchItems(
    existingItems: any[],
    newItems: any[],
    matchProp: string,
    matchPropsOn: string[],
    shouldAppend = true,
  ) {
    const items = Array.isArray(existingItems) ? existingItems : []

    // Find the matching key for this specific property path
    const matchingKey = matchPropsOn.find((key) => {
      const keyPath = key.split('.').slice(0, -1).join('.')

      return keyPath === matchProp
    })

    // If no matching key is configured, simply concatenate the arrays
    if (!matchingKey) {
      return shouldAppend ? [...items, ...newItems] : [...newItems, ...items]
    }

    // Extract the property name we'll use to match items (e.g., 'id' from 'users.data.id')
    const uniqueProperty = matchingKey.split('.').pop() || ''

    // Create a map of new items by their unique property lookups
    const newItemsMap = new Map()

    newItems.forEach((item) => {
      if (this.hasUniqueProperty(item, uniqueProperty)) {
        newItemsMap.set(item[uniqueProperty], item)
      }
    })

    return shouldAppend
      ? this.appendWithMatching(items, newItems, newItemsMap, uniqueProperty)
      : this.prependWithMatching(items, newItems, newItemsMap, uniqueProperty)
  }

  protected appendWithMatching(
    existingItems: any[],
    newItems: any[],
    newItemsMap: Map<any, any>,
    uniqueProperty: string,
  ): any[] {
    // Update existing items with new values, keep non-matching items
    const updatedExisting = existingItems.map((item) => {
      if (this.hasUniqueProperty(item, uniqueProperty) && newItemsMap.has(item[uniqueProperty])) {
        return newItemsMap.get(item[uniqueProperty])
      }

      return item
    })

    // Filter new items to only include those not already in existing items
    const newItemsToAdd = newItems.filter((item) => {
      if (!this.hasUniqueProperty(item, uniqueProperty)) {
        return true // Always add items without unique property
      }

      return !existingItems.some(
        (existing) =>
          this.hasUniqueProperty(existing, uniqueProperty) && existing[uniqueProperty] === item[uniqueProperty],
      )
    })

    return [...updatedExisting, ...newItemsToAdd]
  }

  protected prependWithMatching(
    existingItems: any[],
    newItems: any[],
    newItemsMap: Map<any, any>,
    uniqueProperty: string,
  ): any[] {
    // Filter existing items, keeping only those not being updated
    const untouchedExisting = existingItems.filter((item) => {
      if (this.hasUniqueProperty(item, uniqueProperty)) {
        return !newItemsMap.has(item[uniqueProperty])
      }

      return true
    })

    return [...newItems, ...untouchedExisting]
  }

  protected hasUniqueProperty(item: any, property: string): boolean {
    return item && typeof item === 'object' && property in item
  }

}
export class DeferredReplay {
  requests: any[] = []
  doReload(options: any) { this.requests.push(options) }
  apply(deferred: any) { this.loadDeferredProps(deferred) }
  protected loadDeferredProps(deferred: Page['deferredProps']): void {
    if (deferred) {
      Object.entries(deferred).forEach(([_, group]) => {
        this.doReload({ only: group, deferredProps: true })
      })
    }
  }
}
export function scrollResetReplay(page: Page) {
  current = page
  const state = { component: page.component, loading: true, previousPage: 99, nextPage: 100, lastLoadedPage: 99, requestCount: 3 }
  let resets = 0
  const options = { onReset: () => { resets++ } }
  const getScrollPropFromCurrentPage = () => currentPage.get().scrollProps.posts
  let listener: any
  const router = { on: (_: string, fn: any) => { listener = fn; return () => {} } }
  const resetState = () => {
    const scrollProp = getScrollPropFromCurrentPage()

    state.component = currentPage.get().component
    state.loading = false
    state.previousPage = scrollProp.previousPage
    state.nextPage = scrollProp.nextPage
    state.lastLoadedPage = scrollProp.currentPage
    state.requestCount = 0
  }

  const removeEventListener = router.on('success', (event) => {
    if (state.component === event.detail.page.component && getScrollPropFromCurrentPage().reset) {
      resetState()
      options.onReset?.()
    }
  })

  listener({ detail: { page } })
  return { state, resets }
}
