<template>
  <AppLayout>
    <article class="post" v-if="post">
      <header class="post-header">
        <h1>
          {{ post.Title }}
          <span v-if="post.IsPrivate" class="private-icon"></span>
        </h1>
        <div class="meta">
          <span>{{ formatDate(post.PublishedAt) }}</span>
          <span v-if="isLoggedIn" class="edit-link" style="margin-left: 10px; gap: 5px;">
            | <router-link :to="'/admin/editor/' + post.ID" style="margin-left: 8px;">编辑此文章</router-link>
          </span>
        </div>
      </header>

      <div class="post-content" v-html="post.Body"></div>
    </article>
    <div v-else-if="loading">加载中...</div>
    <div v-else>文章未找到。</div>
  </AppLayout>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue';
import { useRoute } from 'vue-router';
import AppLayout from '@/components/AppLayout.vue';
import api from '@/services/api';
import { authStore } from '@/store';
import Prism from 'prismjs';

const post = ref(null);
const loading = ref(true);
const route = useRoute();
const isLoggedIn = computed(() => authStore.state.isLoggedIn);

const formatDate = (dateString) => {
  const date = new Date(dateString);
  return date.toISOString().split('T')[0];
};

const fetchPost = async (slug) => {
  loading.value = true;
  try {
    const response = await api.getPostBySlug(slug);
    post.value = response.data;
    document.title = post.value.Title;
  } catch (error) {
    console.error('Failed to fetch post:', error);
    post.value = null;
  } finally {
    loading.value = false;
    // Use nextTick to ensure the DOM is updated before highlighting
    await import('vue').then(({ nextTick }) => {
        nextTick(() => {
            Prism.highlightAll();
        });
    });
  }
};

onMounted(() => {
  fetchPost(route.params.slug);
});

watch(() => route.params.slug, (newSlug) => {
  if (newSlug) {
    fetchPost(newSlug);
  }
});
</script>

<style>
/* PrismJS styles are global, so they are not scoped */
@import 'prismjs/themes/prism.css';
</style>