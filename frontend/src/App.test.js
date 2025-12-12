import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import App from './App.vue'

describe('App', () => {
    it('renders "No digital contact list found" when database is empty', async () => {
        // Mock WebSocket
        global.WebSocket = class {
            constructor() {
                this.onmessage = null
                this.onclose = null
                this.close = vi.fn()
                this.send = vi.fn()
            }
        }

        // Mock fetch to return empty data
        global.fetch = vi.fn((url) => {
            if (url.includes('/api/contacts')) {
                return Promise.resolve({
                    json: () => Promise.resolve({ data: [], meta: { total: 0 } })
                })
            }
            if (url.includes('/api/channels')) {
                return Promise.resolve({
                    json: () => Promise.resolve([])
                })
            }
            if (url.includes('/api/zones')) {
                return Promise.resolve({
                    json: () => Promise.resolve([])
                })
            }
            return Promise.resolve({ json: () => Promise.resolve({}) })
        })

        const wrapper = mount(App)

        // Wait for onMounted
        await new Promise(resolve => setTimeout(resolve, 100))

        // Switch to Digital Contact List tab
        const buttons = wrapper.findAll('button')
        const digitalTab = buttons.find(b => b.text().includes('Digital Contact List'))
        await digitalTab.trigger('click')

        // Check for empty state message
        expect(wrapper.text()).toContain('No digital contact list found')
    })
})
