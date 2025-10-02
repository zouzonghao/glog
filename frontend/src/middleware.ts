// src/middleware.ts
import { defineMiddleware } from "astro:middleware";
import { get as getFromCache, set as setInCache } from './utils/cache';

const API_BASE_URL = import.meta.env.PUBLIC_API_URL;

// `context` 和 `next` 会由 Astro 自动提供
export const onRequest = defineMiddleware(async (context, next) => {
  // --- 视图切换逻辑 ---
  const url = context.url;
  const cookies = context.cookies;
  let view: 'list' | 'cards' = 'list'; // 默认视图

  // 1. 检查URL查询参数 (最高优先级)
  const viewFromUrl = url.searchParams.get('view');
  if (viewFromUrl === 'list' || viewFromUrl === 'cards') {
    view = viewFromUrl;
  } else {
    // 2. 检查Cookie (第二优先级)
    const viewFromCookie = cookies.get('view')?.value;
    if (viewFromCookie === 'list' || viewFromCookie === 'cards') {
      view = viewFromCookie;
    } else {
      // 3. 根据设备类型自动选择 (最低优先级)
      const userAgent = context.request.headers.get('user-agent') || '';
      const isMobile = /mobile|android|iphone/i.test(userAgent);
      view = isMobile ? 'cards' : 'list';
    }
  }
  
  // 将最终视图选择存入 context.locals，供页面使用
  context.locals.view = view;

  // --- 认证和路由保护逻辑 ---
  // 默认设置 isLoggedIn 为 false
  context.locals.isLoggedIn = false;
  context.locals.publicSettings = {};

  // --- 全局数据获取 ---
  // 1. 检查认证状态
  const sessionCookie = context.request.headers.get("cookie");
  if (sessionCookie) {
    try {
      const response = await fetch(`${API_BASE_URL}/api/auth/status`, {
        method: "GET",
        headers: { "Cookie": sessionCookie },
      });
      if (response.ok) {
        const data = await response.json();
        context.locals.isLoggedIn = data.isLoggedIn === true;
      }
    } catch (error) {
      console.error("Auth status check failed:", error);
    }
  }

  // 2. 获取公共设置 (带缓存)
  const CACHE_KEY = 'public_settings';
  const CACHE_TTL_SECONDS = 600; // 10 minutes

  let settings = getFromCache(CACHE_KEY);
  if (!settings) {
    try {
      const response = await fetch(`${API_BASE_URL}/api/public-settings`);
      if (response.ok) {
        const data = await response.json();
        settings = data.settings;
        setInCache(CACHE_KEY, settings, CACHE_TTL_SECONDS);
      }
    } catch (error) {
      console.error("Failed to fetch public settings:", error);
    }
  }
  context.locals.publicSettings = settings || {};

  // --- 路由保护 ---
  if (!context.locals.isLoggedIn && context.url.pathname.startsWith('/admin')) {
    // 重定向到登录页面
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