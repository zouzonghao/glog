import { createRouter, createWebHistory } from 'vue-router';
import { authStore } from '@/store';

const routes = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/Home.vue'),
  },
  {
    path: '/post/:slug',
    name: 'PostDetail',
    component: () => import('../views/PostDetail.vue'),
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
  },
  {
    path: '/admin',
    name: 'Admin',
    component: () => import('../views/Admin.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/admin/new',
    name: 'NewPost',
    component: () => import('../views/Editor.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/admin/editor/:id',
    name: 'EditPost',
    component: () => import('../views/Editor.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/admin/settings',
    name: 'Settings',
    component: () => import('../views/Settings.vue'), // Lazy load settings
    meta: { requiresAuth: true }
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, from, next) => {
  authStore.checkAuthStatus(); // Check auth status on every navigation
  if (to.matched.some(record => record.meta.requiresAuth) && !authStore.state.isLoggedIn) {
    next({ name: 'Login' });
  } else {
    next();
  }
});

export default router;