<template>
  <AppLayout>
    <div id="home-page">
      <h2 class="group-title">全部文章</h2>
      <ul class="post-list-minimal">
        <li v-for="post in posts" :key="post.id">
          <span class="date">{{ formatDate(post.publishedAt) }}</span>
          <router-link :to="'/post/' + post.slug" class="title">
            {{ post.title }}
            <span v-if="post.isPrivate" class="private-icon"></span>
          </router-link>
        </li>
        <li v-if="posts.length === 0 && !loading">
          <p>还没有文章。</p>
        </li>
      </ul>
      <div v-if="loading">加载中...</div>
      <Pagination
        v-if="!loading"
        :current-page="pagination.page"
        :total-pages="pagination.totalPages"
        :page-size="pagination.pageSize"
        @page-changed="handlePageChange"
        @pagesize-changed="handlePageSizeChange"
      />
    </div>
  </AppLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import AppLayout from '@/components/AppLayout.vue';
import Pagination from '@/components/Pagination.vue';
import api from '@/services/api';

const posts = ref([]);
const loading = ref(true);
const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
  totalPages: 1,
});

const formatDate = (dateString) => {
  const date = new Date(dateString);
  const year = date.getFullYear();
  const month = ('0' + (date.getMonth() + 1)).slice(-2);
  const day = ('0' + date.getDate()).slice(-2);
  return `${year}年${month}月${day}日`;
};

const fetchPosts = async () => {
  loading.value = true;
  try {
    const response = await api.getPosts(pagination.value.page, pagination.value.pageSize);
    posts.value = response.data.posts;
    pagination.value.total = response.data.total;
    // The public API doesn't return totalPages, so we calculate it
    pagination.value.totalPages = Math.ceil(response.data.total / pagination.value.pageSize);
  } catch (error) {
    console.error('Failed to fetch posts:', error);
  } finally {
    loading.value = false;
  }
};

const handlePageChange = (page) => {
  pagination.value.page = page;
  fetchPosts();
};

const handlePageSizeChange = (pageSize) => {
  pagination.value.page = 1;
  pagination.value.pageSize = pageSize;
  fetchPosts();
};

onMounted(() => {
  fetchPosts();
});
</script>

<style scoped>
/* Styles from index.html can be moved here if they are page-specific */
</style>