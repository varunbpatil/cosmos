import Vue from 'vue'
import App from './App.vue'
import store from './store'
import vuetify from './plugins/vuetify'
import router from './router'
import axios from './axios'
import {
  ConnectorChanges,
  EndpointChanges,
  SyncChanges,
  RunChanges
} from './supabase'

Vue.config.productionTip = false

Vue.prototype.$axios = axios
Vue.prototype.$connectorChanges = ConnectorChanges
Vue.prototype.$endpointChanges = EndpointChanges
Vue.prototype.$syncChanges = SyncChanges
Vue.prototype.$runChanges = RunChanges

new Vue({
  store,
  vuetify,
  router,
  render: h => h(App)
}).$mount('#app')
