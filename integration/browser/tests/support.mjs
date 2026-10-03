import { readFileSync } from 'node:fs'
export const matrix = JSON.parse(readFileSync(new URL('../dist/matrix.json', import.meta.url), 'utf8'))
export async function initialPage(page) {
  return matrix.protocol === 3
    ? JSON.parse(await page.locator('script[data-page="app"]').textContent())
    : JSON.parse(await page.locator('#app').getAttribute('data-page'))
}
export async function propState(page) {
  return JSON.parse(await page.locator('#prop-state').textContent())
}
