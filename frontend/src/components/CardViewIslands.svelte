<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import type { Post, Pagination } from '../services/api';
    import { getPosts, searchPosts } from '../services/api';

    // --- Props ---
    export let initialPosts: Post[] = [];
    export let pagination: Pagination;
    export let searchQuery: string | null = null;

    // --- State for Masonry Columns ---
    let column1Posts: Post[] = [];
    let column2Posts: Post[] = [];

    // --- State for Infinite Scroll ---
    let isLoading = false;
    let hasMore = pagination.has_next;
    let currentPage = pagination.current_page;

    const macaronColors = [
        '#ffb3ba', '#ffdfba', '#ffffba', '#baffc9', '#bae1ff',
        '#fec8d8', '#f2d2a9', '#f9eac3', '#c3e6cb', '#b5d8f2',
        '#f6a6b2', '#e9c39b', '#f4e0a3', '#a9d9c3', '#9cc2e5'
    ];


   // --- Intersection Observer ---
   let sentinel: HTMLDivElement;
   let observer: IntersectionObserver;

    /**
     * Appends a list of posts to the masonry columns.
     */
    function appendPostsToColumns(postsToAppend: Post[]) {
        postsToAppend.forEach(post => {
            if (column1Posts.length <= column2Posts.length) {
                column1Posts = [...column1Posts, post];
            } else {
                column2Posts = [...column2Posts, post];
            }
        });
    }

    /**
     * Fetches the next page of posts.
     */
    async function loadMorePosts() {
        if (isLoading || !hasMore) return;
        isLoading = true;
    
        const nextPage = currentPage + 1;
        const pageSize = pagination.page_size || 10; // Default to 10 if not provided
    
        try {
            // Unified API call logic
            const data = searchQuery
                ? await searchPosts(searchQuery, nextPage, pageSize)
                : await getPosts(nextPage, pageSize);
    
            if (data && data.posts.length > 0) {
                appendPostsToColumns(data.posts);
                currentPage = data.pagination.current_page;
                hasMore = data.pagination.has_next;
            } else {
                hasMore = false;
            }
    
            // Disconnect observer if there are no more posts
            if (!hasMore && observer) {
                observer.disconnect();
            }
        } catch (error) {
            console.error("Failed to load more posts:", error);
            // Optional: Stop trying on error to prevent infinite loops
            hasMore = false;
            if (observer) {
                observer.disconnect();
            }
        } finally {
            isLoading = false;
        }
    }

    /**
     * Handles image loading errors.
     */
    function handleImageError(post: Post) {
        const updateInColumn = (col: Post[]) => {
            const index = col.findIndex(p => p.id === post.id);
            if (index > -1) {
                col[index].cover = '';
                return true;
            }
            return false;
        };

        if (updateInColumn(column1Posts)) {
            column1Posts = [...column1Posts];
        } else if (updateInColumn(column2Posts)) {
            column2Posts = [...column2Posts];
        }
    }
    
    /**
     * Generates a random background color.
     */
    function getRandomColor() {
        return macaronColors[Math.floor(Math.random() * macaronColors.length)];
    }

    // --- Lifecycle ---
    onMount(() => {
        appendPostsToColumns(initialPosts);

        observer = new IntersectionObserver(entries => {
            if (entries[0].isIntersecting) {
                loadMorePosts();
            }
        }, { rootMargin: '200px' });

        // The sentinel element is now guaranteed to exist.
        // We only start observing if there are more posts to load initially.
        if (hasMore) {
            observer.observe(sentinel);
        }
    });

    onDestroy(() => {
        if (observer) {
            observer.disconnect();
        }
    });
</script>

<div class="post-cards-container">
    {#if initialPosts.length > 0}
        <div class="card-column">
            {#each column1Posts as post (post.id)}
                <a href={`/post/${post.slug}`} class="post-card" data-title={post.title}>
                    <div class="card-cover">
                        {#if post.cover}
                            <img src={post.cover} alt={post.title} loading="lazy" on:error={() => handleImageError(post)} />
                        {:else}
                            <div class="pseudo-cover" style="background-color: {getRandomColor()};">
                                <span>{post.title}</span>
                            </div>
                        {/if}
                    </div>
                    <div class="card-info">
                        <h3 class="card-title" style={`view-transition-name: post-${post.slug}`}>
                            {post.title}
                            {#if post.is_private}<span class="private-icon"></span>{/if}
                        </h3>
                    </div>
                </a>
            {/each}
        </div>

        <div class="card-column">
            {#each column2Posts as post (post.id)}
                <a href={`/post/${post.slug}`} class="post-card" data-title={post.title}>
                    <div class="card-cover">
                        {#if post.cover}
                            <img src={post.cover} alt={post.title} loading="lazy" on:error={() => handleImageError(post)} />
                        {:else}
                            <div class="pseudo-cover" style="background-color: {getRandomColor()};">
                                <span>{post.title}</span>
                            </div>
                        {/if}
                    </div>
                    <div class="card-info">
                        <h3 class="card-title" style={`view-transition-name: post-${post.slug}`}>
                            {post.title}
                            {#if post.is_private}<span class="private-icon"></span>{/if}
                        </h3>
                    </div>
                </a>
            {/each}
        </div>
    {:else}
        <p>还没有文章。</p>
    {/if}
</div>

<!-- Infinite Scroll Trigger and Status -->
<div class="infinite-scroll-status">
    {#if isLoading}
        <div class="spinner"></div>
        <p>加载中...</p>
    {:else if !hasMore && initialPosts.length > 0}
        <p class="no-more-posts">没有更多了</p>
    {/if}

    <!-- This is the invisible trigger for the IntersectionObserver -->
    <!-- It is now always rendered to avoid timing issues -->
    <div bind:this={sentinel} class="sentinel"></div>
</div>