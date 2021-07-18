import Vue from 'vue'
import VueRouter from 'vue-router'
import Home from '../views/Home.vue'

Vue.use(VueRouter)

const routes = [
  {
    path: '/',
    name: 'Home',
    component: Home
  },
  {
    path: '/connectors',
    redirect: '/connectors/sources', // redirect to the "sources" child route by default.
    name: 'Connectors',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "connectors" */ '../views/Connectors.vue'),
    children: [
      {
        path: ':type',
        component: () => import(/* webpackChunkName: "connectorsbytype" */ '../views/ConnectorsByType.vue'),
      }
    ]
  },
  {
    path: '/endpoints',
    redirect: '/endpoints/sources',
    name: 'Endpoints',
    component: () => import(/* webpackChunkName: "endpoints" */ '../views/Endpoints.vue'),
    children: [
      {
        path: ':type',
        component: () => import(/* webpackChunkName: "connectorsbytype" */ '../views/EndpointsByType.vue'),
      }
    ]
  },
  {
    path: '/syncs',
    name: 'Syncs',
    component: () => import(/* webpackChunkName: "syncs" */ '../views/Syncs.vue'),
    children: [
      {
        path: ':syncID',
        name: 'Runs',
        component: () => import(/* webpackChunkName: "runs" */ '../views/Runs.vue'),
        children: [
          {
            path: ':runID',
            name: 'Artifacts',
            component: () => import(/* webpackChunkName: "artifacts" */ '../views/Artifacts.vue'),
          }
        ]
      }
    ]
  }
]

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes
})

export default router
