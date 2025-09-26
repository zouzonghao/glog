export interface Post {
    id: number;
    published_at: string;
    title: string;
    slug: string;
    cover: string;
    excerpt: string;
    is_private: boolean;
    content?: string; // content is optional for list items
}

export interface PaginatedPostsResponse {
    posts: Post[];
    pagination: {
        currentPage: number;
        totalPages: number;
        totalRecords: number;
        pageSize: number;
        hasPrev: boolean;
        hasNext: boolean;
    };
}

const API_BASE_URL = import.meta.env.PUBLIC_API_URL;

/**
 * Fetches a paginated list of posts from the API.
 * @param page - The page number to fetch.
 * @param pageSize - The number of posts per page.
 * @returns A promise that resolves to a paginated list of posts.
 */
export async function getPosts(page: number = 1, pageSize: number = 10): Promise<PaginatedPostsResponse> {
    const response = await fetch(`${API_BASE_URL}/api/posts?page=${page}&pageSize=${pageSize}`);
    if (!response.ok) {
        throw new Error('Failed to fetch posts');
    }
    const data = await response.json();
    return data;
}

/**
 * Fetches a single post by its slug.
 * @param slug - The slug of the post to fetch.
 * @returns A promise that resolves to the post detail.
 */
export async function getPostBySlug(slug: string): Promise<Post> {
    const response = await fetch(`${API_BASE_URL}/api/posts/${slug}`);
    if (!response.ok) {
        throw new Error('Post not found');
    }
    const data = await response.json();
    return data;
}

/**
 * Fetches a single post by its ID for editing.
 * @param id - The ID of the post to fetch.
 * @returns A promise that resolves to the post detail.
 */
export async function getPostById(id: string): Promise<Post> {
    const response = await fetch(`${API_BASE_URL}/api/admin/posts/${id}`);
    if (!response.ok) {
        throw new Error('Post not found');
    }
    const data = await response.json();
    return data;
}

/**
 * Searches for posts based on a query.
 * @param query - The search term.
 * @param page - The page number to fetch.
 * @param pageSize - The number of posts per page.
 * @returns A promise that resolves to a paginated list of posts.
 */
export async function searchPosts(query: string, page: number = 1, pageSize: number = 10): Promise<PaginatedPostsResponse> {
    const response = await fetch(`${API_BASE_URL}/api/search?q=${encodeURIComponent(query)}&page=${page}&pageSize=${pageSize}`);
    if (!response.ok) {
        throw new Error('Failed to search posts');
    }
    const data = await response.json();
    return data;
}

/**
 * Checks the authentication status of the user.
 * @returns A promise that resolves to the authentication status.
 */
export async function getAuthStatus(): Promise<{ isLoggedIn: boolean }> {
    const response = await fetch(`${API_BASE_URL}/api/auth/status`, {
        // Include credentials to send session cookies
        credentials: 'include',
    });
    if (!response.ok) {
        // If the request fails, assume the user is not logged in
        return { isLoggedIn: false };
    }
    return response.json();
}

/**
 * Logs out the user.
 * @returns A promise that resolves on successful logout.
 */
export async function logout(): Promise<any> {
    const response = await fetch(`${API_BASE_URL}/api/logout`, {
        method: 'POST', // Use POST for logout as a good practice
        credentials: 'include',
    });
    if (!response.ok) {
        throw new Error('Logout failed');
    }
    return response.json();
}
/**
 * Logs in a user.
 * @param password - The user's password.
 * @returns A promise that resolves on successful login.
 */
export async function login(password: string): Promise<any> {
    const response = await fetch(`${API_BASE_URL}/api/login`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ password }),
    });

    const data = await response.json();

    if (!response.ok) {
        throw new Error(data.message || 'Login failed');
    }

    return data;
}