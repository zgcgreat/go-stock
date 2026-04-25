<script setup>
import {h, onBeforeMount, onBeforeUnmount, onMounted, ref, watch} from "vue";
import {RouterLink, useRouter, useRoute} from 'vue-router'
import {createDiscreteApi, darkTheme, lightTheme, NIcon, NText, NButton, dateZhCN, zhCN} from 'naive-ui'
import {
  AlarmOutline,
  AnalyticsOutline,
  BarChartSharp, Bonfire, BonfireOutline, DiamondOutline, EaselSharp,
  ExpandOutline, Flag,
  Flame, FlameSharp, FlaskOutline, InformationOutline,
  LogoGithub,
  NewspaperOutline,
  NewspaperSharp, Notifications,
  PeopleOutline,
  PowerOutline, Pulse,
  ReorderTwoOutline,
  SettingsOutline, ServerOutline, Skull, SkullOutline, SkullSharp,
  SparklesOutline, FlashOutline,
  StarOutline,
  Wallet, WarningOutline, TimeOutline,
  LogOutOutline
} from '@vicons/ionicons5'
import FloatingAiAssistant from "./components/FloatingAiAssistant.vue";
import FloatingAgentAssistant from "./components/FloatingAgentAssistant.vue";
import {Dragon, Fire, FirefoxBrowser, Gripfire, Robot} from "@vicons/fa";
import {Prompt, ReportAnalytics, ReportMoney, ReportSearch, TrendingUp} from "@vicons/tabler";
import {LocalFireDepartmentRound} from "@vicons/material";
import {AppsList20Regular, BoxSearch20Regular, CommentNote20Filled} from "@vicons/fluent";
import {FireFilled, MoneyCollectOutlined, NotificationFilled, StockOutlined} from "@vicons/antd";
import apiService from './services/api.js'
import {EventsEmit} from './services/wails-bridge.js'

const router = useRouter()
const route = useRoute()
const loading = ref(true)
const loadingMsg = ref("加载数据中...")
const enableNews = ref(false)
const contentStyle = ref("")
const enableFund = ref(false)
const enableAgent = ref(false)
const enableDarkTheme = ref(null)
const content = ref('数据来源于网络,仅供参考;投资有风险,入市需谨慎')
const isFullscreen = ref(false)
const activeKey = ref('stock')
const containerRef = ref({})
const realtimeProfit = ref(0)
const telegraph = ref([])
const groupList = ref([])
const config = ref({})
const isWebMode = ref(true)
const officialStatement = ref("")
const menuOptions = ref([
  {
    label: () =>
        h(
            RouterLink,
            {
              to: {
                name: 'stock',
                query: {
                  groupName: '全部',
                  groupId: 0,
                },
                params: {},
              },
              onClick: () => {
                activeKey.value = 'stock'
              },
            },
            {default: () => '股票自选',}
        ),
    key: 'stock',
    icon: renderIcon(StarOutline),
    children: [
      {
        label: () =>
            h(
                'a',
                {
                  href: '#',
                  type: 'info',
                  onClick: () => {
                    activeKey.value = 'stock'
                    //console.log("push",item)
                    router.push({
                      name: 'stock',
                      query: {
                        groupName: '全部',
                        groupId: 0,
                      },
                    })
                    EventsEmit("changeTab", {ID: 0, name: '全部'})
                  },
                  to: {
                    name: 'stock',
                    query: {
                      groupName: '全部',
                      groupId: 0,
                    },
                  }
                },
                {default: () => '全部',}
            ),
        key: 0,
      }
    ],
  },
  {
    label: () =>
        h(
            RouterLink,
            {
              href: '#',
              to: {
                name: 'market',
                params: {}
              },
              onClick: () => {
                activeKey.value = 'market'
                EventsEmit("changeMarketTab", {ID: 0, name: '市场快讯'})
              },
            },
            {default: () => '市场行情'}
        ),
    key: 'market',
    icon: renderIcon(NewspaperOutline),
    children: [
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "市场快讯",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '市场快讯'})
                  },
                },
                {default: () => '市场快讯',}
            ),
        key: 'market1',
        icon: renderIcon(NewspaperSharp),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "全球股指",
                    },
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '全球股指'})
                  },
                },
                {default: () => '全球股指',}
            ),
        key: 'market2',
        icon: renderIcon(BarChartSharp),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "重大指数",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '重大指数'})
                  },
                },
                {default: () => '重大指数',}
            ),
        key: 'market3',
        icon: renderIcon(AnalyticsOutline),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "行业排名",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '行业排名'})
                  },
                },
                {default: () => '行业排名',}
            ),
        key: 'market4',
        icon: renderIcon(Flag),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "个股资金流向",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '个股资金流向'})
                  },
                },
                {default: () => '个股资金流向',}
            ),
        key: 'market5',
        icon: renderIcon(Pulse),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "龙虎榜",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '龙虎榜'})
                  },
                },
                {default: () => '龙虎榜',}
            ),
        key: 'market6',
        icon: renderIcon(Dragon),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "个股研报",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '个股研报'})
                  },
                },
                {default: () => '个股研报',}
            ),
        key: 'market7',
        icon: renderIcon(StockOutlined),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "公司公告",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '公司公告'})
                  },
                },
                {default: () => '公司公告',}
            ),
        key: 'market8',
        icon: renderIcon(NotificationFilled),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "行业研究",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '行业研究'})
                  },
                },
                {default: () => '行业研究',}
            ),
        key: 'market9',
        icon: renderIcon(ReportSearch),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "当前热门",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '当前热门'})
                  },
                },
                {default: () => '当前热门',}
            ),
        key: 'market10',
        icon: renderIcon(Gripfire),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "指标选股",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '指标选股'})
                  },
                },
                {default: () => '指标选股',}
            ),
        key: 'market11',
        icon: renderIcon(BoxSearch20Regular),
      },
      {
        label: () =>
            h(
                RouterLink,
                {
                  href: '#',
                  to: {
                    name: 'market',
                    query: {
                      name: "名站优选",
                    }
                  },
                  onClick: () => {
                    activeKey.value = 'market'
                    EventsEmit("changeMarketTab", {ID: 0, name: '名站优选'})
                  },
                },
                {default: () => '名站优选',}
            ),
        key: 'market12',
        icon: renderIcon(FirefoxBrowser),
      },
    ]
  },
  {
    label: () =>
        h(
            RouterLink,
            {
              to: {
                name: 'fund',
                query: {
                  name: '基金自选',
                },
              },
              onClick: () => {
                activeKey.value = 'fund'
              },
            },
            {default: () => '基金自选',}
        ),
    show: enableFund.value,
    key: 'fund',
    icon: renderIcon(SparklesOutline),
    children: [
      {
        label: () => h(NText, {type: realtimeProfit.value > 0 ? 'error' : 'success'}, {default: () => '功能完善中！'}),
        key: 'realtimeProfit',
        show: realtimeProfit.value,
        icon: renderIcon(AlarmOutline),
      },
    ]
  },
  {
    label: () =>
        h(
            RouterLink,
            {
              to: {
                name: 'agent',
                query: {
                  name:"Ai智能体",
                },
                onClick: () => {
                  activeKey.value = 'agent'
                },
              }
            },
            {default: () => 'Ai智能体'}
        ),
    key: 'agent',
    show:enableAgent.value,
    icon: renderIcon(Robot),
  },
    {
      label: () =>
          h(
              RouterLink,
              {
                to: {
                  name: 'research',
                  query: {
                    name:"研究中心",
                  },
                },
                onClick: () => {
                  activeKey.value = 'research'
                  setTimeout(() => {
                    EventsEmit("changeResearchTab", {ID: 0, name: 'AI分析报告'})
                  }, 100)
                },
              },
              {default: () => '研究中心'}
          ),
      key: 'research',
      icon: renderIcon(FlaskOutline),
      children:[
          {
            label: () =>
                h(
                    RouterLink,
                    {
                      to: {
                        name: 'research',
                        query: {
                          name:"AI分析报告",
                        },
                      },
                      onClick: () => {
                        activeKey.value = 'research'
                        setTimeout(() => {
                          EventsEmit("changeResearchTab", {ID: 0, name: 'AI分析报告'})
                        }, 100)
                      },
                    },
                    {default: () => 'AI分析报告'}
                ),
            key: 'research1',
            icon: renderIcon(ReportAnalytics),
          },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                      query: {
                        name:"股票推荐记录",
                      },
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 1, name: '股票推荐记录'})
                      }, 100)
                    },
                  },
                  {default: () => '股票推荐记录'}
              ),
          key: 'research2',
          icon: renderIcon(DiamondOutline),
        },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                      query: {
                        name:"异动监控",
                      },
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 2, name: '异动监控'})
                      }, 100)
                    },
                  },
                  {default: () => '异动监控'}
              ),
          key: 'stockChanges',
          icon: renderIcon(TrendingUp),
        },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                      query: {
                        name:"涨停梯队",
                      },
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 9, name: '涨停梯队'})
                      }, 100)
                    },
                  },
                  {default: () => '涨停梯队'}
              ),
          key: 'uplimitLadder',
          icon: renderIcon(LocalFireDepartmentRound),
        },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                      query: {
                        name:"提示词模板",
                      },
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 3, name: '提示词模板'})
                      }, 100)
                    },
                  },
                  {default: () => '提示词模板'}
              ),
          key: 'research3',
          icon: renderIcon(Prompt),
        },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                      query: {
                        name:"股票信息筛选",
                      },
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 3, name: '股票信息筛选'})
                      }, 100)
                    },
                  },
                  {default: () => '股票信息筛选'}
              ),
          key: 'research4',
          icon: renderIcon(AppsList20Regular),
        },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                      query: {
                        name:"定时任务",
                      },
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 5, name: '定时任务'})
                      }, 100)
                    },
                  },
                  {default: () => '定时任务'}
              ),
          key: 'research5',
          icon: renderIcon(TimeOutline),
        },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                      query: {
                        name:"交易日志",
                      },
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 6, name: '交易日志'})
                      }, 100)
                    },
                  },
                  {default: () => '交易日志(beta)'}
              ),
          key: 'research6',
          icon: renderIcon(MoneyCollectOutlined),
        },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 7, name: 'MCP服务'})
                      }, 100)
                    },
                  },
                  {default: () => 'MCP服务'}
              ),
          key: 'mcpServers',
          icon: renderIcon(ServerOutline),
        },
        {
          label: () =>
              h(
                  RouterLink,
                  {
                    to: {
                      name: 'research',
                    },
                    onClick: () => {
                      activeKey.value = 'research'
                      setTimeout(() => {
                        EventsEmit("changeResearchTab", {ID: 8, name: '技能管理'})
                      }, 100)
                    },
                  },
                  {default: () => '技能管理'}
              ),
          key: 'skills',
          icon: renderIcon(FlashOutline),
          show: false,
        },
      ],
    },
  {
    label: () =>
        h(
            RouterLink,
            {
              to: {
                name: 'settings',
                query: {
                  name:"设置",
                },
                onClick: () => {
                  activeKey.value = 'settings'
                },
              }
            },
            {default: () => '设置'}
        ),
    key: 'settings',
    icon: renderIcon(SettingsOutline),
  },
  {
    label: () => h(RouterLink, {
      to: {name: 'userManagement'},
      onClick: () => {
        activeKey.value = 'admin'
      }
    }, {default: () => '用户管理'}),
    key: 'admin',
    show: false,
    icon: renderIcon(PeopleOutline),
  },
  {
    label: () =>
        h(
            RouterLink,
            {
              to: {
                name: 'about',
                query: {
                  name:"关于",
                }
              },
              onClick: () => {
                activeKey.value = 'about'
              },
            },
            {default: () => '关于'}
        ),
    key: 'about',
    icon: renderIcon(LogoGithub),
  },
  {
    show:false,
    label: () => h("a", {
      href: '#',
      onClick: toggleFullscreen,
      title: '全屏 Ctrl+F 退出全屏 Esc',
    }, {default: () => isFullscreen.value ? '取消全屏' : '全屏'}),
    key: 'full',
    icon: renderIcon(ExpandOutline),
  },
  {
    label: () => h("a", {
      href: '#',
      onClick: WindowHide,
      title: '隐藏到托盘区 Ctrl+Z',
    }, {default: () => '隐藏到托盘区'}),
    key: 'hide',
    icon: renderIcon(ReorderTwoOutline),
  },
  // {
  //   label: ()=> h("a", {
  //     href: 'javascript:void(0)',
  //     style: 'cursor: move;',
  //     onClick: toggleStartMoveWindow,
  //   }, { default: () => '移动' }),
  //   key: 'move',
  //   icon: renderIcon(MoveOutline),
  // },
  {
    label: () => h("a", {
      href: '#',
      onClick: handleLogout,
    }, {default: () => '退出登录'}),
    key: 'logout',
    icon: renderIcon(LogOutOutline),
  },
  {
    label: () => h("a", {
      href: '#',
      onClick: Quit,
    }, {default: () => '退出程序'}),
    key: 'exit',
    icon: renderIcon(PowerOutline),
  },
])

function renderIcon(icon) {
  return () => h(NIcon, null, {default: () => h(icon)})
}

function toggleFullscreen(e) {
  activeKey.value = 'full'
  if (isFullscreen.value) {
    if (document.exitFullscreen) document.exitFullscreen()
  } else {
    document.documentElement.requestFullscreen()
  }
  isFullscreen.value = !isFullscreen.value
}

function handleLogout() {
  localStorage.removeItem('token')
  router.push('/login')
}

function handleQuit() {
  if (isWebMode.value) {
    window.close()
  }
}

// 加载配置
async function loadConfig() {
  if (!localStorage.getItem('token')) return
  try {
    const res = await apiService.getUserSettings()
    if (res && res.data) {
      config.value = res.data
      enableFund.value = res.data.enableFund !== false
      enableAgent.value = res.data.enableAgent !== false
      enableNews.value = res.data.enableNews !== false
      if (res.data.darkTheme) {
        enableDarkTheme.value = darkTheme
      } else {
        enableDarkTheme.value = null
      }
    }
  } catch (e) {
    console.warn('Failed to load config:', e)
    enableFund.value = true
    enableAgent.value = true
    enableNews.value = true
  }
}

// 加载分组列表
async function loadGroupList() {
  if (!localStorage.getItem('token')) return
  try {
    const res = await apiService.client.get('/groups')
    if (res.data && res.data.code === 0) {
      const data = res.data.data
      const list = Array.isArray(data) ? data : (data?.list || [])
      groupList.value = list
      if (list.length > 0) {
        updateMenuWithGroups()
      }
    }
  } catch (e) {
    console.warn('Failed to load groups:', e)
  }
}

function updateMenuWithGroups() {
  menuOptions.value.forEach((item) => {
    if (item.key === 'stock') {
      const existingKeys = (item.children || []).map(c => c.key)
      (groupList.value || []).forEach(g => {
        if (!existingKeys.includes(g.ID)) {
          item.children.push({
            label: () => h('a', {
              href: '#',
              type: 'info',
              onClick: () => {
                router.push({name: 'stock', query: {groupName: g.name, groupId: g.ID}})
                setTimeout(() => {
                  window.dispatchEvent(new CustomEvent('changeTab', {detail: g}))
                }, 100)
              },
            }, {default: () => g.name}),
            key: g.ID,
          })
        }
      })
    }
  })
}

// 监听路由变化更新 activeKey
watch(() => route.name, (newName) => {
  if (newName) activeKey.value = newName
})

onBeforeMount(async () => {
  officialStatement.value = "go-stock Web版"
  content.value = officialStatement.value + "\n\n" + content.value

  await Promise.all([loadConfig(), loadGroupList()])

  // 设置普通菜单项的显示状态
  menuOptions.value.forEach((item) => {
    if (item.key === 'fund') item.show = enableFund.value
    if (item.key === 'agent') item.show = enableAgent.value
  })

  // 检查用户权限以决定是否显示用户管理菜单
  if (localStorage.getItem('token')) {
    try {
      const userProfile = await apiService.getUserProfile()
      console.log('[DEBUG] API完整响应:', userProfile)
      console.log('[DEBUG] userProfile.data:', userProfile.data)
      
      // axios返回的response.data就是后端返回的数据
      const userData = userProfile.data || userProfile
      console.log('[DEBUG] 实际用户数据:', userData)
      
      if (userData) {
        const role = userData.role
        console.log('[DEBUG] 用户角色:', role)
        // admin 或 super_admin 可以访问用户管理
        if (role === 'admin' || role === 'super_admin') {
          console.log('[DEBUG] 显示用户管理菜单')
          // 找到用户管理菜单项并设为显示
          const adminMenuIndex = menuOptions.value.findIndex(item => item.key === 'admin')
          if (adminMenuIndex !== -1) {
            // 使用splice触发响应式更新
            const updatedMenu = [...menuOptions.value]
            updatedMenu[adminMenuIndex] = { ...updatedMenu[adminMenuIndex], show: true }
            menuOptions.value = updatedMenu
            console.log('[DEBUG] 菜单项已设置为显示, 当前菜单:', menuOptions.value[adminMenuIndex])
          }
        } else {
          console.log('[DEBUG] 用户角色不是admin或super_admin,不显示菜单')
        }
      }
    } catch (error) {
      console.error('[ERROR] 获取用户信息失败:', error)
      // 出错时不显示用户管理菜单
    }
  }

  loading.value = false
  loadingMsg.value = "加载完成..."
})

onMounted(() => {
  contentStyle.value = "max-height: calc(92vh);overflow: hidden"
})

onBeforeUnmount(() => {
  // Clean up any event listeners
})
</script>

<template>
  <n-config-provider ref="containerRef" :theme="enableDarkTheme" :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-notification-provider>
        <n-modal-provider>
          <n-dialog-provider>
            <n-watermark
                :content="''"
                cross
                selectable
                :font-size="16"
                :line-height="16"
                :width="500"
                :height="400"
                :x-offset="50"
                :y-offset="150"
                :rotate="-15"
            >
              <FloatingAgentAssistant />
              <n-flex>
                <n-grid x-gap="12" :cols="1">
                  <n-gi>
                    <n-spin :show="loading">
                      <template #description>
                        {{ loadingMsg }}
                      </template>
                      <n-marquee :speed="100" style="position: relative;top:0;z-index: 19;width: 100%"
                                 v-if="(telegraph.length>0)&&(enableNews)">
                        <n-tag type="warning" v-for="item in telegraph" style="margin-right: 10px">
                          {{ item }}
                        </n-tag>
                      </n-marquee>
                      <n-scrollbar :style="contentStyle">
                        <n-skeleton v-if="loading" height="calc(100vh)" />
                        <RouterView/>
                      </n-scrollbar>
                    </n-spin>
                  </n-gi>
                  <n-gi style="position: fixed;bottom:0;z-index: 9;width: 100%;">
                    <n-card size="small" style="--wails-draggable:no-drag">
                      <n-menu style="font-size: 18px;"
                              v-model:value="activeKey"
                              mode="horizontal"
                              :options="menuOptions"
                              responsive
                      />
                    </n-card>
                  </n-gi>
                </n-grid>
              </n-flex>
            </n-watermark>
          </n-dialog-provider>
        </n-modal-provider>
      </n-notification-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<style>
</style>