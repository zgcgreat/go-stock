import {createRouter, createWebHashHistory} from 'vue-router'

import stockView from '../components/stock.vue'
import settingsView from '../components/settings.vue'
import aboutView from "../components/about.vue";
import fundView from "../components/fund.vue";
import marketView from "../components/market.vue";
import agentChat from "../components/agent-chat.vue"
import research from "../components/researchIndex.vue";
import cronTaskManager from "../components/cron-task-manager.vue"
import mcpServerManager from "../components/mcp-server-manager.vue"
import loginView from "../components/Login.vue"
import registerView from "../components/Register.vue"
import userManagementView from "../components/UserManager.vue"

const routes = [
    { path: '/login', component: loginView, name: 'login', meta: { public: true } },
    { path: '/register', component: registerView, name: 'register', meta: { public: true } },
    { path: '/', component: stockView,name: 'stock'},
    { path: '/fund', component: fundView,name: 'fund' },
    { path: '/settings', component: settingsView,name: 'settings' },
    { path: '/about', component: aboutView,name: 'about' },
    { path: '/market', component: marketView,name: 'market' },
    { path: '/agent', component: agentChat,name: 'agent' },
    { path: '/research', component: research,name: 'research' },
    { path: '/cron-tasks', component: cronTaskManager,name: 'cronTasks' },
    { path: '/mcp-servers', component: mcpServerManager,name: 'mcpServers' },
    { path: '/user-management', component: userManagementView,name: 'userManagement' },
]

const router = createRouter({
    history: createWebHashHistory(),
    routes,
})

router.beforeEach((to, from, next) => {
    const token = localStorage.getItem('token')
    if (!to.meta.public && !token) {
        sessionStorage.setItem('redirectAfterLogin', to.fullPath)
        next({ name: 'login' })
    } else if (to.meta.public && token && (to.name === 'login' || to.name === 'register')) {
        next({ path: '/' })
    } else {
        next()
    }
})

export default router
