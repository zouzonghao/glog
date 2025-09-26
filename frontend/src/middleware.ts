// src/middleware.ts
import { defineMiddleware } from "astro:middleware";

const API_BASE_URL = import.meta.env.PUBLIC_API_URL;

// `context` 和 `next` 会由 Astro 自动提供
export const onRequest = defineMiddleware(async (context, next) => {
  // 默认设置 isLoggedIn 为 false
  context.locals.isLoggedIn = false;

  // 从浏览器请求中获取 cookie
  const sessionCookie = context.request.headers.get("cookie");

  // 如果没有 cookie，直接进入下一个中间件或页面渲染
  if (!sessionCookie) {
    return next();
  }

  // 如果有 cookie，将其转发到后端 API 进行验证
  try {
    const response = await fetch(`${API_BASE_URL}/api/auth/status`, {
      method: "GET",
      headers: {
        // 将浏览器发送的 cookie 原样转发给后端
        "Cookie": sessionCookie,
      },
    });

    if (response.ok) {
      const data = await response.json();
      // 根据后端的响应更新 isLoggedIn 状态
      context.locals.isLoggedIn = data.isLoggedIn === true;
    }
  } catch (error) {
    // 如果后端 API 请求失败，保持未登录状态
    console.error("Auth status check failed:", error);
    context.locals.isLoggedIn = false;
  }

  // 路由保护逻辑
  // 如果用户未登录但试图访问 /admin/ 路径
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
  return next();
});