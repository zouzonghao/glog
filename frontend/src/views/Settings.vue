<template>
  <AppLayout>
    <div class="settings-container">
      <div class="setting-header">
        <h2 class="group-title">站点信息</h2>
      </div>
      <form @submit.prevent="saveSiteInfo" class="app-form">
        <div class="settings-form-group settings-form-group-spaced">
          <label for="password">管理员密码</label>
          <input type="password" v-model="settings.password" placeholder="留空则不修改" autocomplete="new-password">
        </div>
        <div class="settings-form-group">
          <label for="favicon">浏览器标签页图标（url）</label>
          <input type="text" v-model="settings.favicon" autocomplete="no">
        </div>
        <div class="settings-form-group">
          <label for="site_title">站点标题</label>
          <input type="text" v-model="settings.site_title" autocomplete="no">
        </div>
        <div class="settings-form-group">
          <label for="site_description">站点描述</label>
          <input type="text" v-model="settings.site_description" autocomplete="no">
        </div>
        <div class="settings-form-group">
          <label for="cover_prefix">封面前缀</label>
          <input type="text" v-model="settings.cover_prefix" autocomplete="no">
        </div>
        <div class="settings-actions settings-form-group-spaced">
          <button type="submit" class="btn">💾 保存站点信息</button>
        </div>
      </form>

      <div class="setting-header setting-header-separated">
        <h2 class="group-title">AI 集成</h2>
      </div>
      <div class="backup-actions settings-form-group-spaced">
        <button @click="openModal('ai')" class="btn">🔧 基本模型</button>
        <button @click="openModal('imageApi')" class="btn">🎨 AI 封面模型</button>
        <button @click="openModal('aiLogs')" class="btn">📜 AI 日志</button>
      </div>

      <div class="setting-header setting-header-separated">
        <h2 class="group-title">备份与恢复</h2>
      </div>
      <div class="backup-actions settings-form-group-spaced">
        <button @click="openModal('github')" class="btn">🔧 GitHub 备份</button>
        <button @click="openModal('webdav')" class="btn">🔧 WebDAV 备份</button>
        <a href="/api/backup" class="btn">💾 下载备份</a>
        <form class="upload-form-wrapper">
          <input type="file" @change="handleFileUpload" accept=".zip,.json" class="hidden-file-input" ref="fileInput">
          <button type="button" @click="triggerUpload" class="btn">📤 上传恢复</button>
        </form>
      </div>
    </div>
  </AppLayout>

  <!-- Modals -->
  <Modal :show="activeModal === 'ai'" @close="closeModal">
    <h3>AI 功能设置</h3>
    <form id="ai-settings-form" class="app-form" autocomplete="off">
      <div class="settings-form-group">
        <label for="openai_base_url">OpenAI Compatible Base URL</label>
        <input type="text" id="openai_base_url" name="openai_base_url" v-model="settings.openai_base_url" autocomplete="no">
      </div>
      <div class="settings-form-group">
        <label for="openai_token">OpenAI Compatible Token</label>
        <input type="password" id="openai_token" name="openai_token" placeholder="留空则不修改" autocomplete="new-password">
      </div>
      <div class="settings-form-group">
        <label for="openai_model">OpenAI Compatible 模型</label>
        <input type="text" id="openai_model" name="openai_model" v-model="settings.openai_model" autocomplete="no">
      </div>
      <div class="modal-actions">
        <button type="button" class="btn">🚀 测试连接</button>
        <button type="button" @click="saveSettings" class="btn">💾 保存设置</button>
      </div>
    </form>
  </Modal>

  <Modal :show="activeModal === 'imageApi'" @close="closeModal">
    <h3>AI 封面功能设置 (ImageAPI)</h3>
    <form class="app-form" autocomplete="off">
      <div class="settings-form-group">
        <label for="imageapi_url">ImageAPI URL</label>
        <input type="text" v-model="settings.imageapi_url" autocomplete="no">
      </div>
      <div class="settings-form-group">
        <label for="imageapi_token">ImageAPI Token</label>
        <input type="password" name="imageapi_token" placeholder="留空则不修改" autocomplete="new-password">
      </div>
      <div class="settings-form-group">
        <label for="imageapi_model">ImageAPI 模型</label>
        <input type="text" v-model="settings.imageapi_model" autocomplete="no">
      </div>
      <div class="modal-actions">
        <button type="button" class="btn">🚀 测试连接</button>
        <button type="button" @click="saveSettings" class="btn">💾 保存设置</button>
      </div>
    </form>
  </Modal>

  <Modal :show="activeModal === 'aiLogs'" @close="closeModal">
    <h3>AI 任务日志</h3>
    <div class="logs-container">
      <pre>正在加载日志...</pre>
    </div>
    <div class="modal-actions">
      <button type="button" class="btn">🗑️ 清除日志</button>
    </div>
  </Modal>

  <Modal :show="activeModal === 'github'" @close="closeModal">
    <h3>GitHub 备份设置</h3>
    <form class="app-form" autocomplete="off">
      <div class="settings-form-group">
        <label for="github_repo">仓库名 (格式: user/repo)</label>
        <input type="text" v-model="settings.github_repo" autocomplete="no">
      </div>
      <div class="settings-form-group">
        <label for="github_branch">分支名</label>
        <input type="text" v-model="settings.github_branch" autocomplete="no">
      </div>
      <div class="settings-form-group">
        <label for="github_token">Personal Access Token</label>
        <input type="password" name="github_token" placeholder="留空则不修改" autocomplete="new-password">
      </div>
      <div class="settings-form-group">
        <label for="github_interval">备份间隔（小时），0 表示不启用</label>
        <input type="number" v-model="settings.github_interval" min="0" autocomplete="no">
      </div>
      <div class="modal-actions">
        <button type="button" class="btn">⚡ 立即备份</button>
        <button type="button" class="btn">🚀 测试连接</button>
        <button type="button" @click="saveSettings" class="btn">💾 保存设置</button>
      </div>
    </form>
  </Modal>

  <Modal :show="activeModal === 'webdav'" @close="closeModal">
    <h3>WebDAV 备份设置</h3>
    <form class="app-form" autocomplete="off">
      <div class="settings-form-group">
        <label for="webdav_url">服务器地址</label>
        <input type="text" v-model="settings.webdav_url" autocomplete="no">
      </div>
      <div class="settings-form-group">
        <label for="webdav_user">账号</label>
        <input type="text" v-model="settings.webdav_user" autocomplete="no">
      </div>
      <div class="settings-form-group">
        <label for="webdav_password">密码</label>
        <input type="password" name="webdav_password" placeholder="留空则不修改" autocomplete="new-password">
      </div>
      <div class="settings-form-group">
        <label for="webdav_interval">备份间隔（小时），0 表示不启用</label>
        <input type="number" v-model="settings.webdav_interval" min="0" autocomplete="no">
      </div>
      <div class="modal-actions">
        <button type="button" class="btn">⚡ 立即备份</button>
        <button type="button" class="btn">🚀 测试连接</button>
        <button type="button" @click="saveSettings" class="btn">💾 保存设置</button>
      </div>
    </form>
  </Modal>

</template>

<script setup>
import { ref, onMounted } from 'vue';
import AppLayout from '@/components/AppLayout.vue';
import Modal from '@/components/Modal.vue';
import api from '@/services/api';
import { showNotification } from '@/services/notification';

const settings = ref({});
const fileInput = ref(null);
const activeModal = ref(null);

const openModal = (modalName) => {
  activeModal.value = modalName;
};

const closeModal = () => {
  activeModal.value = null;
};

const saveSettings = async () => {
  try {
    await api.updateSettings(settings.value);
    showNotification('设置已保存！', 'success');
    closeModal();
  } catch (error) {
    console.error('Failed to save settings:', error);
    showNotification('保存失败！', 'error');
  }
};

const fetchSettings = async () => {
  try {
    const response = await api.getSettings();
    settings.value = response.data.settings;
  } catch (error) {
    console.error('Failed to fetch settings:', error);
  }
};

const saveSiteInfo = async () => {
  // This function now can be merged into saveSettings or kept separate
  // For now, let's just call the main save function
  await saveSettings();
};

const triggerUpload = () => {
  fileInput.value.click();
};

const handleFileUpload = async (event) => {
  const file = event.target.files[0];
  if (!file) return;

  const formData = new FormData();
  formData.append('backup', file);
  
  // You might need a password prompt here
  const password = prompt("请输入备份文件密码 (如果是加密的):");
  if (password) {
      formData.append('password', password);
  }

  try {
    await api.uploadBackup(formData);
    showNotification('恢复成功！', 'success');
  } catch (error) {
    console.error('Upload failed:', error);
    showNotification('恢复失败！', 'error');
  }
};

onMounted(fetchSettings);
</script>

<style scoped>
.settings-container {
  max-width: 700px;
  margin: 0 auto;
  padding: 20px;
}
.hidden-file-input {
  display: none;
}
</style>