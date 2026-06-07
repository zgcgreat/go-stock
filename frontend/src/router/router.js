import {createMemoryHistory, createRouter, createWebHashHistory, createWebHistory} from 'vue-router'

import stockView from '../components/stock.vue'
import settingsView from '../components/settings.vue'
import aboutView from "../components/about.vue";
import fundView from "../components/fund.vue";
import marketView from "../components/market.vue";
import agentChat from "../components/agent-chat.vue"
import research from "../components/researchIndex.vue";
import cronTaskManager from "../components/cron-task-manager.vue"
import mcpServerManager from "../components/mcp-server-manager.vue"
import klineAnalysis from "../components/kline-analysis.vue"
import userManager from "../components/UserManager.vue"
import loginView from "../components/Login.vue"
import registerView from "../components/Register.vue"
import Auth from '../utils/auth.js'

const routes = [
    { path: '/login', component: loginView, name: 'login', meta: { public: true } },
    { path: '/register', component: registerView, name: 'register', meta: { public: true } },
    { path: '/', component: stockView, name: 'stock'},
    { path: '/fund', component: fundView, name: 'fund' },
    { path: '/settings', component: settingsView, name: 'settings' },
    { path: '/about', component: aboutView, name: 'about' },
    { path: '/market', component: marketView, name: 'market' },
    { path: '/agent', component: agentChat, name: 'agent' },
    { path: '/research', component: research, name: 'research' },
    { path: '/cron-tasks', component: cronTaskManager, name: 'cronTasks' },
    { path: '/mcp-servers', component: mcpServerManager, name: 'mcpServers' },
    { path: '/kline-analysis', component: klineAnalysis, name: 'klineAnalysis' },
    { path: '/user-management', component: userManager, name: 'userManagement' },

]

const router = createRouter({
    //history: createWebHistory(),
    history: createWebHashHistory(),
    routes,
})

// Web 模式下路由守卫：未登录跳转 /login
router.beforeEach((to, from, next) => {
    // 桌面端 Wails 模式不需要登录检查
    if (window.go && window.go.main && window.go.main.App) {
        next()
        return
    }
    // Web 模式：公开路由直接放行
    if (to.meta?.public) {
        next()
        return
    }
    // Web 模式：需要登录的路由
    if (!Auth.isLoggedIn()) {
        next({ name: 'login', query: { redirect: to.fullPath } })
        return
    }
    next()
})

export default router