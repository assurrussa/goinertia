// ESM resolution uses this profile's peer dependencies, never the v2 parent.
export const resolve = (specifier) => import.meta.resolve(specifier)
