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
import dailyOperationPlanView from "../components/DailyOperationPlan.vue"
import aiConfigManagerView from "../components/ai-config-manager.vue"
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
    { path: '/daily-operation-plans', component: dailyOperationPlanView, name: 'dailyOperationPlans' },
    { path: '/ai-config-manager', component: aiConfigManagerView, name: 'aiConfigManager' },

]

const router = createRouter({
    //history: createWebHistory(),
    history: createWebHashHistory(),
    routes,
})

/**
 * 检查 JWT token 是否过期
 * @param {string} token
 * @returns {boolean} true=已过期或无法解析, false=未过期
 */
function isTokenExpired(token) {
    if (!token) return true
    try {
        const parts = token.split('.')
        if (parts.length !== 3) return true
        const payload = JSON.parse(atob(parts[1]))
        if (!payload.exp) return false  // 没有 exp 字段，不判断过期
        // exp 是秒级时间戳，预留 30 秒缓冲
        return payload.exp * 1000 < Date.now() + 30000
    } catch (e) {
        return true  // 解析失败视为过期
    }
}

// 上次远程验证 token 的时间戳，避免每次导航都请求
let lastTokenCheckTime = 0
const TOKEN_CHECK_INTERVAL = 5 * 60 * 1000  // 5 分钟内不重复远程校验

// Web 模式下路由守卫：未登录或 token 过期时跳转 /login
router.beforeEach(async (to, from, next) => {
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
    const token = Auth.getToken()
    if (!token) {
        next({ name: 'login', query: { redirect: to.fullPath } })
        return
    }

    // JWT 本地过期检查（快速，无需网络请求）
    if (isTokenExpired(token)) {
        Auth.clearToken()
        window.dispatchEvent(new Event('user-info-updated'))
        next({ name: 'login', query: { redirect: to.fullPath } })
        return
    }

    // token 未过期：首次进入或距离上次校验超过 5 分钟时，远程验证一次
    const userInfo = Auth.getUserInfo()
    const needRemoteCheck = !userInfo || (Date.now() - lastTokenCheckTime > TOKEN_CHECK_INTERVAL)
    if (needRemoteCheck) {
        try {
            const res = await fetch('/api/v1/user/profile', {
                headers: { Authorization: `Bearer ${token}` }
            })
            if (res.ok) {
                const data = await res.json()
                if (data.data) {
                    Auth.setUserInfo(data.data)
                }
                lastTokenCheckTime = Date.now()
                next()
            } else {
                // token 无效（被撤销等），清除并跳转登录
                Auth.clearToken()
                window.dispatchEvent(new Event('user-info-updated'))
                next({ name: 'login', query: { redirect: to.fullPath } })
            }
        } catch (e) {
            // 网络错误时放行（不因网络问题阻止页面访问）
            next()
        }
    } else {
        next()
    }
})

export default router