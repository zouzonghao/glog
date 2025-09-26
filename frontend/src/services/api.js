import axios from 'axios';

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  withCredentials: true, // Send cookies with requests
});

// You can add interceptors here for global error handling or auth tokens

export default {
  // Post APIs
  getPosts(page = 1, pageSize = 10) {
    return apiClient.get('/posts', { params: { page, pageSize } });
  },
  getPostBySlug(slug) {
    return apiClient.get(`/posts/${slug}`);
  },
  
  // Auth APIs
  login(password) {
    return apiClient.post('/login', { password });
  },
  logout() {
    return apiClient.post('/logout');
  },

  // Admin APIs
  getAdminPosts(params) {
    return apiClient.get('/admin/posts', { params });
  },
  getPostById(id) {
    return apiClient.get(`/admin/posts/${id}`);
  },
  createPost(postData) {
    return apiClient.post('/posts', postData);
  },
  updatePost(id, postData) {
    return apiClient.put(`/posts/${id}`, postData);
  },
  deletePost(id) {
    return apiClient.delete(`/posts/${id}`);
  },
  batchUpdatePosts(data) {
    return apiClient.post('/posts/batch-update', data);
  },

  // Settings APIs
  getSettings() {
    return apiClient.get('/settings');
  },
  updateSettings(settings) {
    return apiClient.post('/settings', settings);
  },
  uploadBackup(formData) {
    return apiClient.post('/backup/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
};