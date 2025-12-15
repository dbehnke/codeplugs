import { createRouter, createWebHistory } from 'vue-router'

// Views
import ChannelsView from '../views/ChannelsView.vue'
import ZonesView from '../views/ZonesView.vue'
import ScanListView from '../views/ScanListView.vue'
import DMRTalkgroupsView from '../views/DMRTalkgroupsView.vue'
import DMRContactsView from '../views/DMRContactsView.vue'
import NXDNTalkgroupsView from '../views/NXDNTalkgroupsView.vue'
import NXDNContactsView from '../views/NXDNContactsView.vue'

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [
        {
            path: '/',
            redirect: '/channels'
        },
        {
            path: '/channels',
            name: 'channels',
            component: ChannelsView
        },
        {
            path: '/zones',
            name: 'zones',
            component: ZonesView
        },
        {
            path: '/scanlists',
            name: 'scanlists',
            component: ScanListView
        },
        {
            path: '/dmr-talkgroups',
            name: 'dmr-talkgroups',
            component: DMRTalkgroupsView
        },
        {
            path: '/dmr-contacts',
            name: 'dmr-contacts',
            component: DMRContactsView
        },
        {
            path: '/nxdn-talkgroups',
            name: 'nxdn-talkgroups',
            component: NXDNTalkgroupsView
        },
        {
            path: '/nxdn-contacts',
            name: 'nxdn-contacts',
            component: NXDNContactsView
        }
    ]
})

export default router
