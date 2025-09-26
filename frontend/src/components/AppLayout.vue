<template>
  <div class="main-wrapper">
    <header class="main-header">
      <div class="header-left">
        <input type="checkbox" id="menu-toggle" class="nav-toggle-checkbox">
        <label for="menu-toggle" class="menu-toggle" aria-label="Toggle menu">
          <div class="hamburger-box">
            <div class="hamburger-inner"></div>
          </div>
        </label>
        <div class="site-info">
          <router-link to="/" class="site-title">{{ siteTitle }}</router-link>
          <nav class="main-nav" id="main-nav">
            <ul>
              <li><router-link to="/">主页</router-link></li>
              <template v-if="isLoggedIn">
                <li><router-link to="/admin/new">新建</router-link></li>
                <li><router-link to="/admin">管理</router-link></li>
                <li><router-link to="/admin/settings">设置</router-link></li>
                <li><a href="#" @click.prevent="logout">登出</a></li>
              </template>
              <template v-else>
                <li><router-link to="/login">登录</router-link></li>
              </template>
            </ul>
          </nav>
        </div>
      </div>
      <div class="header-right">
        <form @submit.prevent="performSearch" class="search-form">
          <input type="search" name="q" placeholder="搜索..." class="search-input" v-model="searchQuery">
          <button type="submit" class="search-button" aria-label="Search">
            <img src="@/assets/pic/search.png" alt="Search" class="search-icon">
          </button>
        </form>
        <a href="#" @click.prevent="toggleTheme" class="theme-toggle" title="Toggle theme">
          <svg class="sun-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>
          <svg class="moon-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path></svg>
        </a>
      </div>
    </header>

    <main>
      <slot></slot>
    </main>

    <footer class="main-footer">
      <p>&copy; {{ siteTitle }}. Powered by Glog.</p>
    </footer>
  </div>
  <a href="#top" id="back-to-top" class="back-to-top" title="Back to top">&uarr;</a>
  <div id="notification-container"></div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { authStore } from '@/store';
import api from '@/services/api';

// Mock data, will be replaced by API calls
const siteTitle = ref('Glog');
const isLoggedIn = computed(() => authStore.state.isLoggedIn);
const searchQuery = ref('');

const router = useRouter();

const performSearch = () => {
  if (searchQuery.value.trim()) {
    router.push({ path: '/search', query: { q: searchQuery.value } });
  }
};

const logout = async () => {
  try {
    await api.logout();
    authStore.logout();
    router.push('/');
  } catch (error) {
    console.error('Logout failed:', error);
  }
};

const toggleTheme = () => {
  const currentTheme = document.documentElement.classList.contains('dark') ? 'dark' : 'light';
  const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
  document.documentElement.classList.remove(currentTheme);
  document.documentElement.classList.add(newTheme);
  localStorage.setItem('theme', newTheme);
};

onMounted(() => {
  // Logic from base.html script tag
  const savedTheme = localStorage.getItem("theme");
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  let theme;

  if (savedTheme) {
    theme = savedTheme;
  } else if (prefersDark) {
    theme = "dark";
  } else {
    theme = "light";
  }
  
  if (!document.documentElement.classList.contains(theme)) {
    const otherTheme = theme === 'dark' ? 'light' : 'dark';
    if (document.documentElement.classList.contains(otherTheme)) {
        document.documentElement.classList.remove(otherTheme);
    }
    document.documentElement.classList.add(theme);
  }
  if (!savedTheme) {
      localStorage.setItem("theme", theme);
  }

  // Back to top button logic
  const backToTop = document.getElementById('back-to-top');
  if (backToTop) {
    window.addEventListener('scroll', () => {
      if (window.scrollY > 200) {
        backToTop.style.display = 'block';
      } else {
        backToTop.style.display = 'none';
      }
    });
  }
});
</script>

<style scoped>
/* Scoped styles if needed, but most styles are global */
</style>