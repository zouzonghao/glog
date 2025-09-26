# 在 Glog 项目中设置 Astro 视图过渡（View Transitions）指南

本文档详细说明了为 Glog Astro 前端项目添加页面间平滑过渡动画（View Transitions）的完整步骤和遇到的问题解决方案。

## 什么是视图过渡？

视图过渡（View Transitions）是现代浏览器提供的一项新功能，允许开发者在单页应用（SPA）或多页应用（MPA）的页面状态变化之间创建流畅的动画效果。Astro 通过内置的 `<ClientRouter />` 组件（前身为 `<ViewTransitions />`）极大地简化了这一过程，为多页应用带来了媲美单页应用的流畅体验。

---

## 实施步骤

我们在项目中通过以下四个核心步骤实现了视图过渡功能：

### 步骤 1：全局启用客户端路由

这是激活视图过渡功能的基础。

- **文件**: [`frontend/src/layouts/BaseLayout.astro`](frontend/src/layouts/BaseLayout.astro)
- **操作**:
  1.  从 `astro:transitions` 导入 `<ClientRouter />` 组件。
  2.  将该组件放置在主布局文件的 `<head>` 标签内。

```astro
---
// frontend/src/layouts/BaseLayout.astro
import { ClientRouter } from 'astro:transitions';
---
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8">
    <title>{title} | {siteTitle}</title>
    ...
    <ClientRouter />
  </head>
  <body>
    ...
  </body>
</html>
```

**注意**: 在 Astro 5.0 版本中，`ViewTransitions` 组件已被重命名为 `ClientRouter`，以更准确地反映其在客户端路由中的作用。功能保持不变。

### 步骤 2：标记需要过渡的元素

为了让 Astro 知道哪些元素在不同页面间是“同一个”并需要应用动画，我们使用了 `transition:name` 指令。

- **文件**:
  - [`frontend/src/components/PostCard.astro`](frontend/src/components/PostCard.astro) (列表/卡片中的标题)
  - [`frontend/src/pages/post/[slug].astro`](frontend/src/pages/post/[slug].astro) (文章详情页的标题)
- **操作**:
  为起始页面和目标页面上对应的元素赋予一个**唯一的、相同的** `transition:name`。我们使用文章的 `slug` 来确保其唯一性。

**示例 (PostCard.astro):**
```astro
// frontend/src/components/PostCard.astro
<a href={`/post/${post.slug}`} ...>
  ...
  <h3 class="card-title" style={`view-transition-name: post-${post.slug}`}>
    {post.title}
  </h3>
  ...
</a>
```

**示例 ([slug].astro):**
```astro
// frontend/src/pages/post/[slug].astro
<article class="post">
  <header class="post-header">
    <h1 style={`view-transition-name: post-${slug}`}>
      {post.title}
    </h1>
    ...
  </header>
  ...
</article>
```
通过这种方式，当用户点击卡片跳转时，浏览器会自动为这两个标题元素创建平滑的“变形”动画。我们同样为导航栏的搜索框设置了固定的 `transition:name`，使其在页面跳转时也能保持位置。

### 步骤 3：持久化客户端脚本控制的元素

对于完全由客户端 JavaScript 控制状态的元素（例如我们的主题切换按钮），直接进行过渡可能会导致状态错乱。我们需要让 Astro 在过渡期间“保持”这些元素不变。

- **文件**: [`frontend/src/layouts/BaseLayout.astro`](frontend/src/layouts/BaseLayout.astro)
- **操作**: 为主题切换按钮添加 `transition:persist` 指令。

```astro
// frontend/src/layouts/BaseLayout.astro
<a href="#" id="theme-toggle" class="theme-toggle" title="Toggle theme" transition:persist>
  <svg class="sun-icon" ...></svg>
  <svg class="moon-icon" ...></svg>
</a>
```
这告诉 Astro 在页面切换时不替换这个 `<a>` 元素，而是将其从旧页面直接移动到新页面，从而保留其 DOM 状态。

### 步骤 4：适配 Astro 的页面生命周期

仅仅持久化元素是不够的，因为页面过渡不是传统的整页刷新，**页面加载脚本不会重新运行**。这会导致事件监听器丢失，功能失效。

- **文件**: [`frontend/src/scripts/main.ts`](frontend/src/scripts/main.ts)
- **操作**: 利用 Astro 提供的 `astro:after-swap` 事件，在每次页面过渡完成后重新初始化我们的脚本。

**重构前:**
```typescript
// 只在首次加载时运行
document.addEventListener('DOMContentLoaded', function() {
  // ... 设置主题切换、登出等事件监听器
});
```

**重构后:**
```typescript
// frontend/src/scripts/main.ts

// 1. 将所有初始化逻辑封装到一个函数中
function initializePage() {
  // ... 设置主题切换、登出等事件监听器
  // (注意处理重复绑定的问题)
}

// 2. 在两个不同的生命周期事件中调用它
document.addEventListener('DOMContentLoaded', initializePage); // 首次加载
document.addEventListener('astro:after-swap', initializePage);   // 每次页面过渡后
```
这是解决客户端脚本在 Astro 视图过渡中失效问题的**最终且最优雅的方案**。

---

## 总结

通过以上步骤，我们成功地为项目集成了现代化、流畅且健壮的视图过渡功能，显著提升了用户在页面间导航时的视觉体验。