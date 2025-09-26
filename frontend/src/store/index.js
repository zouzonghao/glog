import { reactive, readonly } from 'vue';

const state = reactive({
  isLoggedIn: false,
});

const checkAuthStatus = async () => {
  // This is a placeholder. In a real app, you'd have an endpoint
  // like /api/auth/status to check if the session is valid.
  // For now, we can check for a cookie or a flag from a previous login.
  // A simple approach is to assume not logged in on page load,
  // and only set to true after a successful login.
  state.isLoggedIn = sessionStorage.getItem('isLoggedIn') === 'true';
};

const login = () => {
  state.isLoggedIn = true;
  sessionStorage.setItem('isLoggedIn', 'true');
};

const logout = () => {
  state.isLoggedIn = false;
  sessionStorage.removeItem('isLoggedIn');
};

// Initialize auth status on app load
checkAuthStatus();

export const authStore = {
  state: readonly(state),
  login,
  logout,
  checkAuthStatus,
};