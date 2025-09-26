import { createApp } from 'vue'
import './assets/css/style.css'
import './assets/css/prism.css'
import './assets/css/cards.css'
import App from './App.vue'
import router from './router'

const app = createApp(App)
app.use(router)
app.mount('#app')
