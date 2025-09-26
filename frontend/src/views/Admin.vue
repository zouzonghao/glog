<template>
  <AppLayout>
    <div class="admin-container">
      <div class="admin-header">
        <h2 class="group-title">文章管理</h2>
        <form @submit.prevent="performSearch" class="search-form admin-search-form">
          <input type="search" v-model="searchQuery" placeholder="搜索文章..." class="search-input">
          <button type="submit" class="search-button" aria-label="Search">
            <img src="@/assets/pic/search.png" alt="Search" class="search-icon">
          </button>
        </form>
      </div>

      <div class="post-list-container">
        <div class="post-list-header">
          <div class="col-checkbox"><input type="checkbox" v-model="selectAll" @change="toggleSelectAll"></div>
          <div class="col-title">标题</div>
          <div class="col-private">私密</div>
          <div class="col-date">发布日期</div>
          <div class="col-actions">操作</div>
        </div>
        <div class="post-list-body">
          <div v-for="post in posts" :key="post.id" class="post-list-item">
            <div class="col-checkbox"><input type="checkbox" v-model="selectedPosts" :value="post.id"></div>
            <div class="col-title" :title="post.title">
              <router-link :to="'/post/' + post.slug">{{ post.title }}</router-link>
            </div>
            <div class="col-private">{{ post.isPrivate ? '是' : '-' }}</div>
            <div class="col-date">{{ formatDate(post.publishedAt) }}</div>
            <div class="col-actions">
              <router-link :to="'/admin/editor/' + post.id">[编辑]</router-link>
              <div class="delete-wrapper" tabindex="0">
                <span class="delete-init" @click="confirmDelete(post.id)">[删除]</span>
              </div>
            </div>
          </div>
          <div v-if="posts.length === 0 && !loading" class="empty-state">
            <p>{{ searchQuery ? `没有找到与 "${searchQuery}" 相关的文章。` : '没有文章。' }}</p>
          </div>
        </div>
      </div>

      <div class="batch-actions-container">
        <button @click="batchUpdate('delete')" :disabled="selectedPosts.length === 0" class="btn">批量删除</button>
        <button @click="batchUpdate('private')" :disabled="selectedPosts.length === 0" class="btn">设为私密</button>
        <button @click="batchUpdate('public')" :disabled="selectedPosts.length === 0" class="btn">设为公开</button>
      </div>
      
      <Pagination
        :current-page="pagination.page"
        :total-pages="pagination.totalPages"
        :page-size="pagination.pageSize"
        @page-changed="handlePageChange"
        @pagesize-changed="handlePageSizeChange"
      />

      <div v-if="showModal" id="modal-container" class="modal-container" style="display: flex;">
        <div class="modal-content">
          <p id="modal-text">{{ modalText }}</p>
          <div class="modal-actions">
            <button @click="executeDelete" class="modal-btn confirm">确认</button>
            <button @click="cancelDelete" class="modal-btn cancel">取消</button>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue';
import AppLayout from '@/components/AppLayout.vue';
import Pagination from '@/components/Pagination.vue';
import api from '@/services/api';

const posts = ref([]);
const loading = ref(true);
const searchQuery = ref('');
const selectedPosts = ref([]);
const showModal = ref(false);
const modalText = ref('');
let postToDelete = null;
let batchAction = null;

const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
  totalPages: 1,
});

const selectAll = computed({
  get: () => posts.value.length > 0 && selectedPosts.value.length === posts.value.length,
  set: (value) => {
    if (value) {
      selectedPosts.value = posts.value.map(p => p.id);
    } else {
      selectedPosts.value = [];
    }
  }
});

const formatDate = (dateString) => new Date(dateString).toISOString().split('T')[0];

const fetchPosts = async () => {
  loading.value = true;
  try {
    const params = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      q: searchQuery.value,
    };
    const response = await api.getAdminPosts(params);
    posts.value = response.data.posts;
    pagination.value.total = response.data.total;
    pagination.value.totalPages = response.data.totalPages;
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

const performSearch = () => {
  pagination.value.page = 1;
  fetchPosts();
};

const confirmDelete = (id) => {
  postToDelete = id;
  batchAction = null;
  modalText.value = '确定要删除这篇文章吗？';
  showModal.value = true;
};

const batchUpdate = (action) => {
  batchAction = action;
  postToDelete = null;
  if (action === 'delete') {
    modalText.value = `确定要删除选中的 ${selectedPosts.value.length} 篇文章吗？`;
  } else if (action === 'private') {
    modalText.value = `确定要将选中的 ${selectedPosts.value.length} 篇文章设为私密吗？`;
  } else {
    modalText.value = `确定要将选中的 ${selectedPosts.value.length} 篇文章设为公开吗？`;
  }
  showModal.value = true;
};

const executeDelete = async () => {
  if (postToDelete) {
    try {
      await api.deletePost(postToDelete);
      posts.value = posts.value.filter(p => p.id !== postToDelete);
    } catch (error) {
      console.error('Failed to delete post:', error);
    }
  } else if (batchAction) {
    try {
      const isPrivate = batchAction === 'private';
      await api.batchUpdatePosts({
        ids: selectedPosts.value,
        action: batchAction === 'delete' ? 'delete' : 'privacy',
        is_private: isPrivate,
      });
      if (batchAction === 'delete') {
        posts.value = posts.value.filter(p => !selectedPosts.value.includes(p.id));
      } else {
        await fetchPosts(); // Re-fetch to show changes
      }
      selectedPosts.value = [];
    } catch (error) {
      console.error('Batch action failed:', error);
    }
  }
  cancelDelete();
};

const cancelDelete = () => {
  showModal.value = false;
  postToDelete = null;
  batchAction = null;
};

onMounted(() => fetchPosts());
</script>

<style scoped>
.admin-container {
  max-width: 900px;
  margin: 0 auto;
  padding: 20px;
}
</style>