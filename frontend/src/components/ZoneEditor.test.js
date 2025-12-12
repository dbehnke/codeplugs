
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import ZoneEditor from './ZoneEditor.vue'

describe('ZoneEditor', () => {
    const mockChannels = [
        { ID: 1, Name: 'Channel A' },
        { ID: 2, Name: 'Channel B' },
        { ID: 3, Name: 'Channel C' }
    ]

    const mockZone = {
        ID: 1,
        Name: 'Test Zone',
        Channels: [
            { ID: 2, Name: 'Channel B' } // Starts with B
        ]
    }

    it('renders available and member channels correctly', () => {
        const wrapper = mount(ZoneEditor, {
            props: {
                modelValue: mockZone,
                allChannels: mockChannels
            }
        })

        // Available should contain A and C (ID 1, 3) because B is in Zone
        const available = wrapper.findAll('.available-list .channel-item')
        expect(available.length).toBe(2)
        expect(available[0].text()).toContain('Channel A')
        expect(available[1].text()).toContain('Channel C')

        // Members should contain B
        const members = wrapper.findAll('.member-list .channel-item')
        expect(members.length).toBe(1)
        expect(members[0].text()).toContain('Channel B')
    })

    it('adds a channel to the zone', async () => {
        const wrapper = mount(ZoneEditor, {
            props: {
                modelValue: { ...mockZone, Channels: [] }, // Start empty
                allChannels: mockChannels
            }
        })

        // Select Channel A in available
        await wrapper.find('.available-list .channel-item').trigger('click')
        // Click Add (>>)
        await wrapper.find('.btn-add').trigger('click')

        // Verify emitted event updates model
        // Note: ZoneEditor likely emits update:modelValue or we check internal state if we test interactions
        // Let's assume it updates local state and emits on save, OR emits update:modelValue immediately.
        // For editor modal, usually we have local state and Save emits.
        // Let's check visual state first (moved to right)

        const members = wrapper.findAll('.member-list .channel-item')
        expect(members.length).toBe(1)
        expect(members[0].text()).toContain('Channel A')
    })

    it('reorders channels', async () => {
        const zoneWithTwo = {
            ID: 1, Name: 'Test',
            Channels: [
                { ID: 1, Name: 'A' },
                { ID: 2, Name: 'B' }
            ]
        }
        const wrapper = mount(ZoneEditor, {
            props: { modelValue: zoneWithTwo, allChannels: mockChannels }
        })

        // Select 'B' (index 1)
        const members = wrapper.findAll('.member-list .channel-item')
        await members[1].trigger('click') // Select B

        // Click Up
        await wrapper.find('.btn-up').trigger('click')

        // Expect B to be first now
        const newMembers = wrapper.findAll('.member-list .channel-item')
        expect(newMembers[0].text()).toContain('B')
        expect(newMembers[1].text()).toContain('A')
    })
})
