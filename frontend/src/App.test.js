import { mount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import App from './App.vue'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'

// Mock Sidebar and RouterView components to avoid deep dependencies
vi.mock('./components/Sidebar.vue', () => ({
    default: {
        template: '<div id="sidebar"><button>Digital Contact List</button></div>'
    }
}))

const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/', component: { template: '<div>Dashboard</div>' } }]
})

describe('App', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
    })

    it('renders sidebar and transition area', async () => {
        // Mock WebSocket
        global.WebSocket = class {
            constructor() {
                this.onmessage = null
                this.onclose = null
                this.close = vi.fn()
                this.send = vi.fn()
            }
        }

        const wrapper = mount(App, {
            global: {
                plugins: [router]
            }
        })

        await router.push('/')
        await router.isReady()
        await flushPromises()

        expect(wrapper.find('#sidebar').exists()).toBe(true)
        expect(wrapper.text()).toContain('Dashboard')
    })
})
