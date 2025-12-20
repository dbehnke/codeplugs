import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RoamingChannelsView from './RoamingChannelsView.vue'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'

// Mock global fetch
global.fetch = vi.fn()

const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/roaming/channels', component: RoamingChannelsView }]
})

describe('RoamingChannelsView.vue', () => {
    beforeEach(() => {
        vi.resetAllMocks()
        setActivePinia(createPinia())
    })

    it('fetches and displays roaming channels on mount', async () => {
        const mockChannels = [
            { ID: 1, Name: 'Roam 1', RxFrequency: 446.0, ColorCode: 1, TimeSlot: 1 },
            { ID: 2, Name: 'Roam 2', RxFrequency: 447.0, ColorCode: 2, TimeSlot: 2 }
        ]

        fetch.mockImplementation((url) => {
            if (url === '/api/roaming/channels') {
                return Promise.resolve({
                    ok: true,
                    json: async () => ({ success: true, data: mockChannels })
                })
            }
            return Promise.resolve({ ok: true, json: async () => ({ success: true, data: [] }) })
        })

        const wrapper = mount(RoamingChannelsView, {
            global: {
                plugins: [router]
            }
        })

        await flushPromises()

        expect(wrapper.text()).toContain('Roam 1')
        expect(wrapper.text()).toContain('Roam 2')
        expect(wrapper.text()).toContain('446')
    })
})
