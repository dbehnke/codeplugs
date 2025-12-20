import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RoamingZonesView from './RoamingZonesView.vue'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'

// Mock global fetch
global.fetch = vi.fn()

const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/roaming/zones', component: RoamingZonesView }]
})

describe('RoamingZonesView.vue', () => {
    beforeEach(() => {
        vi.resetAllMocks()
        setActivePinia(createPinia())
    })

    it('fetches and displays roaming zones on mount', async () => {
        const mockZones = [
            { ID: 1, Name: 'North Zone', Channels: [] },
            { ID: 2, Name: 'South Zone', Channels: [] }
        ]

        fetch.mockImplementation((url) => {
            if (url === '/api/roaming/zones') {
                return Promise.resolve({
                    ok: true,
                    json: async () => ({ success: true, data: mockZones })
                })
            }
            if (url === '/api/roaming/channels') {
                return Promise.resolve({ ok: true, json: async () => ({ success: true, data: [] }) })
            }
            return Promise.resolve({ ok: true, json: async () => ({ success: true, data: [] }) })
        })

        const wrapper = mount(RoamingZonesView, {
            global: {
                plugins: [router]
            }
        })

        await flushPromises()

        expect(wrapper.text()).toContain('North Zone')
        expect(wrapper.text()).toContain('South Zone')
    })
})
