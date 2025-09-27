export interface Post {
    id: number;
    published_at: string;
    title: string;
    slug: string;
    cover: string;
    excerpt: string;
    is_private: boolean;
}

// For single post responses, we get the full post object from the backend
export interface PostDetail extends Post {
	content_html: string;
}

// For editor, we need the raw markdown content
export interface PostForEditor extends Post {
    content: string;
}

export interface PaginatedPostsResponse {
    posts: Post[];
    pagination: {
    	current_page: number;
    	   total_pages: number;
    	   total_records: number;
    	   page_size: number;
    	   has_prev: boolean;
    	   has_next: boolean;
    };
}

const API_BASE_URL = import.meta.env.PUBLIC_API_URL;

/**
 * Custom error class for API fetch errors.
 * Contains the HTTP status code for more specific error handling.
 */
export class ApiError extends Error {
    status: number;

    constructor(message: string, status: number) {
        super(message);
        this.name = 'ApiError';
        this.status = status;
    }
}

/**
 * A wrapper for the native fetch function that includes credentials,
 * handles API errors, and automatically redirects to the login page
 * on 401 Unauthorized responses.
 *
 * It also handles passing cookies during Server-Side Rendering (SSR).
 * @param url - The URL to fetch.
 * @param options - The options for the fetch request.
 * @param cookies - Optional Astro.cookies object for SSR requests.
 * @returns A promise that resolves to the JSON response.
 */
async function apiFetch(url: string, options: RequestInit = {}, cookies?: any): Promise<any> {
    const defaultOptions: RequestInit = {
        credentials: 'include', // Always send cookies in browser
        headers: {
            'Content-Type': 'application/json',
            ...options.headers,
        },
        ...options,
    };

    // If running on the server (SSR) and cookies are provided, forward them.
    if (import.meta.env.SSR && cookies) {
        const sessionCookie = cookies.get('glog_session')?.value;
        if (sessionCookie) {
            (defaultOptions.headers as Record<string, string>)['Cookie'] = `glog_session=${sessionCookie}`;
        }
    }

    const response = await fetch(url, defaultOptions);

    if (response.status === 401) {
        // If we are on the server, we can't redirect. Throwing an error is enough.
        if (import.meta.env.SSR) {
            throw new ApiError('Unauthorized', 401);
        }
        // If we are already on the login page, don't redirect.
        if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
            window.location.href = '/login';
        }
        // Throw an error to stop the current execution chain.
        throw new ApiError('Unauthorized', 401);
    }

    // For other errors, try to parse the JSON body for a message.
    if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: `HTTP error! Status: ${response.status}` }));
        throw new ApiError(errorData.message || 'An unknown error occurred', response.status);
    }

    // If the response is successful, parse and return the JSON.
    return response.json();
}


/**
 * Fetches a paginated list of posts from the API.
 * @param page - The page number to fetch.
 * @param pageSize - The number of posts per page.
 * @returns A promise that resolves to a paginated list of posts.
 */
export async function getPosts(page: number = 1, pageSize: number = 10, cookies?: any): Promise<PaginatedPostsResponse> {
    return apiFetch(`${API_BASE_URL}/api/posts?page=${page}&pageSize=${pageSize}`, {}, cookies);
}

/**
 * Fetches a single post by its slug.
 * @param slug - The slug of the post to fetch.
 * @returns A promise that resolves to the post detail.
 */
export async function getPostBySlug(slug: string, cookies?: any): Promise<PostDetail> {
	return apiFetch(`${API_BASE_URL}/api/posts/${slug}`, {}, cookies);
}

/**
 * Fetches a single post by its ID for editing.
 * @param id - The ID of the post to fetch.
 * @returns A promise that resolves to the post detail.
 */
export async function getPostById(id: string, cookies?: any): Promise<PostForEditor> {
    const response = await apiFetch(`${API_BASE_URL}/api/admin/posts/${id}`, {}, cookies);
    return response.post;
}

/**
 * Searches for posts based on a query.
 * @param query - The search term.
 * @param page - The page number to fetch.
 * @param pageSize - The number of posts per page.
 * @returns A promise that resolves to a paginated list of posts.
 */
export async function searchPosts(query: string, page: number = 1, pageSize: number = 10, cookies?: any): Promise<PaginatedPostsResponse> {
    return apiFetch(`${API_BASE_URL}/api/search?q=${encodeURIComponent(query)}&page=${page}&pageSize=${pageSize}`, {}, cookies);
}

/**
 * Fetches a paginated list of posts for the admin panel. Requires authentication.
 * @param page - The page number to fetch.
 * @param pageSize - The number of posts per page.
 * @param query - Optional search query.
 * @param cookies - The Astro.cookies object for SSR.
 * @returns A promise that resolves to a paginated list of all posts.
 */
export async function getAdminPosts(page: number = 1, pageSize: number = 10, query: string = '', cookies?: any): Promise<PaginatedPostsResponse> {
    const params = new URLSearchParams({
        page: page.toString(),
        pageSize: pageSize.toString(),
    });
    if (query) {
        params.set('q', query);
    }
    return apiFetch(`${API_BASE_URL}/api/admin/posts?${params.toString()}`, {}, cookies);
}

/**
 * Checks the authentication status of the user.
 * This function is a bit special as a 401 is an expected outcome.
 * @returns A promise that resolves to the authentication status.
 */
export async function getAuthStatus(): Promise<{ isLoggedIn: boolean }> {
    try {
        // We don't use apiFetch here because a 401 is not an exception in this case.
        const response = await fetch(`${API_BASE_URL}/api/auth/status`, {
            credentials: 'include',
        });
        if (!response.ok) {
            return { isLoggedIn: false };
        }
        return response.json();
    } catch (error) {
        console.error("Auth status check failed:", error);
        return { isLoggedIn: false };
    }
}

/**
 * Logs out the user.
 * @returns A promise that resolves on successful logout.
 */
export async function logout(): Promise<any> {
    return apiFetch(`${API_BASE_URL}/api/logout`, {
        method: 'POST',
    });
}
/**
 * Logs in a user.
 * @param password - The user's password.
 * @returns A promise that resolves on successful login.
 */
export async function login(password: string): Promise<any> {
    // This call doesn't need the 401 redirect logic from apiFetch.
    const response = await fetch(`${API_BASE_URL}/api/login`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ password }),
        credentials: 'include',
    });

    const data = await response.json();

    if (!response.ok) {
        throw new Error(data.message || 'Login failed');
    }

    return data;
}

/**
 * Fetches the site settings. Requires authentication.
 * @param cookies - The Astro.cookies object for SSR.
 * @returns A promise that resolves to the site settings.
 */
export async function getSettings(cookies?: any): Promise<any> {
    const response = await apiFetch(`${API_BASE_URL}/api/settings`, {}, cookies);
    return response.settings;
}