import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ScanLists from './ScanListView.vue'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'

// Mock global fetch
global.fetch = vi.fn()

const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/scanlists', component: ScanLists }]
})

describe('ScanListView.vue', () => {
    beforeEach(() => {
        vi.resetAllMocks()
        setActivePinia(createPinia())
    })

    it('fetches and displays scan lists on mount', async () => {
        const mockChannels = []
        const mockLists = [
            { ID: 1, Name: 'List A', Channels: [] },
            { ID: 2, Name: 'List B', Channels: [] }
        ]

        // Mock fetch for channels and scanlists
        fetch.mockImplementation((url) => {
            if (url === '/api/channels') {
                return Promise.resolve({
                    ok: true,
                    json: async () => mockChannels
                })
            }
            if (url === '/api/scanlists') {
                return Promise.resolve({
                    ok: true,
                    json: async () => mockLists
                })
            }
            return Promise.reject(new Error('Unknown URL'))
        })

        const wrapper = mount(ScanLists, {
            global: {
                plugins: [router]
            }
        })

        // Wait for fetch
        await flushPromises()

        // Check text content
        expect(wrapper.text()).toContain('List A')
        expect(wrapper.text()).toContain('List B')
    })

    it('displays empty state when no lists', async () => {
        fetch.mockImplementation((url) => {
            if (url === '/api/channels') return Promise.resolve({ ok: true, json: async () => [] })
            if (url === '/api/scanlists') return Promise.resolve({ ok: true, json: async () => [] })
            return Promise.reject(new Error('Unknown URL'))
        })

        const wrapper = mount(ScanLists, {
            global: {
                plugins: [router]
            }
        })

        await flushPromises()
        expect(wrapper.text()).toContain('Select a list to edit or create a new one.')
    })
})
