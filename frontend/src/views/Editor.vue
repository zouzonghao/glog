<template>
  <AppLayout>
    <div class="editor-container">
      <h2 class="group-title">{{ isNewPost ? '文章新建' : '文章编辑' }}</h2>
      <form @submit.prevent="savePost" class="app-form">
        <input type="hidden" v-model="post.id">
        
        <div class="editor-form-group editor-title-group">
          <input type="text" v-model="post.title" required>
          <input type="text" v-model="post.published_at">
        </div>
        
        <div class="editor-form-group">
          <textarea v-model="post.content" rows="20"></textarea>
        </div>
        
        <div v-if="post.ai_cover" class="editor-form-group">
          <textarea v-model="post.ai_cover_prompt" rows="2" placeholder="AI封面提示词（留空则根据文章内容自动生成）"></textarea>
        </div>
        
        <div class="editor-options">
          <div class="form-group-inline">
            <input type="checkbox" id="ai_summary" v-model="post.ai_summary">
            <label for="ai_summary">AI摘要</label>
          </div>
          <div class="form-group-inline">
            <input type="checkbox" id="ai_cover" v-model="post.ai_cover">
            <label for="ai_cover">AI封面</label>
          </div>
          <div class="form-group-inline">
            <input type="checkbox" id="is_private" v-model="post.is_private">
            <label for="is_private">私密</label>
          </div>
        </div>

        <div class="editor-actions">
          <button type="submit" :disabled="saving" class="btn btn-editor-action">💾 {{ saving ? '保存中...' : '保存文章' }}</button>
          <router-link v-if="post.slug" :to="'/post/' + post.slug" class="btn btn-editor-action open-post-link">🔗 打开文章</router-link>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import AppLayout from '@/components/AppLayout.vue';
import api from '@/services/api';

const route = useRoute();
const router = useRouter();
const saving = ref(false);

const post = ref({
  id: 0,
  title: '未命名标题',
  content: '\n\n<!--more-->\n',
  published_at: new Date().toISOString().slice(0, 16).replace('T', ' '),
  is_private: false,
  ai_summary: false,
  ai_cover: false,
  ai_cover_prompt: '',
  slug: '',
});

const isNewPost = computed(() => !route.params.id);

const fetchPostData = async (id) => {
  try {
    const response = await api.getPostById(id);
    const fetchedPost = response.data.post;
    post.value = {
        ...fetchedPost,
        id: fetchedPost.ID,
        title: fetchedPost.Title,
        content: fetchedPost.Content,
        published_at: new Date(fetchedPost.PublishedAt).toISOString().slice(0, 16).replace('T', ' '),
        is_private: fetchedPost.IsPrivate,
        slug: fetchedPost.Slug,
        // AI fields might not be present, provide defaults
        ai_summary: fetchedPost.ai_summary || false,
        ai_cover: fetchedPost.ai_cover || false,
        ai_cover_prompt: fetchedPost.ai_cover_prompt || '',
    };
  } catch (error) {
    console.error('Failed to fetch post data:', error);
    // Redirect or show error
  }
};

const savePost = async () => {
  saving.value = true;
  try {
    const payload = {
      ...post.value,
      // Ensure correct format for Go backend
      published_at: post.value.published_at,
    };
    
    let response;
    if (isNewPost.value) {
      response = await api.createPost(payload);
    } else {
      response = await api.updatePost(post.value.id, payload);
    }

    if (response.data.status === 'success') {
      // If it's a new post, the router needs to be pushed to the new ID
      if (isNewPost.value) {
        router.push({ name: 'EditPost', params: { id: response.data.post_id } });
      } else {
        // Maybe just show a success notification
        alert('文章已保存！');
      }
    }
  } catch (error) {
    console.error('Failed to save post:', error);
  } finally {
    saving.value = false;
  }
};

onMounted(() => {
  if (!isNewPost.value) {
    fetchPostData(route.params.id);
  }
});
</script>