import { createApp } from 'vue'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import App from './App.vue'
import router from './router'
import { applyTheme, getTheme } from '@/stores/theme'
import './style.css'

dayjs.locale('zh-cn')
applyTheme(getTheme())

createApp(App).use(router).use(Antd).mount('#app')
