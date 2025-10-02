// src/middleware.ts
import { defineMiddleware } from "astro:middleware";

const API_BASE_URL = import.meta.env.PUBLIC_API_URL;

// `context` 和 `next` 会由 Astro 自动提供
export const onRequest = defineMiddleware(async (context, next) => {
  // --- 1. 绝对安全的初始化 ---
  // 无论后续发生什么，locals 都有一个基本结构
  context.locals.view = 'list';
  context.locals.isLoggedIn = false;
  context.locals.publicSettings = {};

  // --- 2. 视图切换逻辑 (无副作用，可以安全执行) ---
  const url = context.url;
  const cookies = context.cookies;
  let view: 'list' | 'cards' = 'list';
  const viewFromUrl = url.searchParams.get('view');
  if (viewFromUrl === 'list' || viewFromUrl === 'cards') {
    view = viewFromUrl;
  } else {
    const viewFromCookie = cookies.get('view')?.value;
    if (viewFromCookie === 'list' || viewFromCookie === 'cards') {
      view = viewFromCookie;
    } else {
      const userAgent = context.request.headers.get('user-agent') || '';
      const isMobile = /mobile|android|iphone/i.test(userAgent);
      view = isMobile ? 'cards' : 'list';
    }
  }
  context.locals.view = view;

  // --- 3. 全局数据获取 (包裹在单个 try/catch 中, 移除了缓存逻辑) ---
  try {
    if (!API_BASE_URL) {
      console.error("[Middleware] PUBLIC_API_URL is not set. Gracefully skipping API calls.");
    } else {
      // 检查认证状态
      const sessionCookie = context.request.headers.get("cookie");
      if (sessionCookie) {
        const authResponse = await fetch(`${API_BASE_URL}/api/auth/status`, {
          method: "GET",
          headers: { "Cookie": sessionCookie },
        });
        if (authResponse.ok) {
          const authData = await authResponse.json();
          context.locals.isLoggedIn = authData.isLoggedIn === true;
        }
      }

      // 获取公共设置 (无缓存)
      const settingsResponse = await fetch(`${API_BASE_URL}/api/public-settings`);
      if (settingsResponse.ok) {
        const settingsData = await settingsResponse.json();
        if (settingsData && typeof settingsData.settings === 'object' && settingsData.settings !== null) {
          context.locals.publicSettings = settingsData.settings;
        } else {
           console.warn("[Middleware] API response for public-settings is OK but malformed. Data:", settingsData);
        }
      } else {
        console.error(`[Middleware] Failed to fetch public settings. Status: ${settingsResponse.status}`);
      }
    }
  } catch (error) {
    console.error("[Middleware] A critical error occurred during data fetching:", error);
  }

  // --- 4. 路由保护 ---
  if (!context.locals.isLoggedIn && context.url.pathname.startsWith('/admin')) {
    return context.redirect('/login');
  }

  // 如果用户已登录但试图访问 /login 页面
  if (context.locals.isLoggedIn && context.url.pathname === '/login') {
    // 重定向到管理后台首页
    return context.redirect('/admin');
  }

  // 处理完毕，继续执行后续操作
  const response = await next();

  // 在响应中设置 'view' Cookie，实现状态持久化
  // 仅当从URL参数中明确设置了视图时，才更新Cookie
  if (viewFromUrl === 'list' || viewFromUrl === 'cards') {
    context.cookies.set('view', view, {
      path: '/',
      maxAge: 60 * 60 * 24 * 365, // 1 year
    });
  }
  
  return response;
});