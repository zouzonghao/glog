<template>
  <AppLayout>
    <div class="editor-container login-container">
      <form @submit.prevent="handleLogin" class="editor-form app-form">
        <div class="form-group login-form-group">
          <input type="password" v-model="password" required autocomplete="current-password" placeholder="请输入密码" class="login-password-input">
          <button type="submit" :disabled="loading" class="login-submit-btn">
            {{ loading ? '登录中...' : '登录' }}
          </button>
        </div>
        <p v-if="error" class="error-message">{{ error }}</p>
      </form>
    </div>
  </AppLayout>
</template>

<script setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import AppLayout from '@/components/AppLayout.vue';
import api from '@/services/api';
import { authStore } from '@/store';

const password = ref('');
const loading = ref(false);
const error = ref('');
const router = useRouter();

const handleLogin = async () => {
  loading.value = true;
  error.value = '';
  try {
    const response = await api.login(password.value);
    if (response.data.status === 'success') {
      authStore.login();
      router.push('/admin');
    } else {
      error.value = response.data.message || '登录失败';
    }
  } catch (err) {
    error.value = err.response?.data?.message || '发生错误，请重试';
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.error-message {
  color: red;
  text-align: center;
  margin-top: 10px;
}
</style>