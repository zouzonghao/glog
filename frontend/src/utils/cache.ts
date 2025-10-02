interface CacheEntry<T> {
  data: T;
  expiry: number;
}

// This Map will persist in the Astro server's memory
const cache = new Map<string, CacheEntry<any>>();

/**
 * Sets a value in the cache.
 * @param key The cache key.
 * @param data The data to cache.
 * @param ttlSeconds Time-to-live in seconds.
 */
export function set<T>(key: string, data: T, ttlSeconds: number): void {
  const expiry = Date.now() + ttlSeconds * 1000;
  cache.set(key, { data, expiry });
}

/**
 * Gets a value from the cache.
 * @param key The cache key.
 * @returns The cached data if it exists and has not expired, otherwise null.
 */
export function get<T>(key: string): T | null {
  const entry = cache.get(key);
  if (!entry) {
    return null;
  }

  // Check for expiry
  if (Date.now() > entry.expiry) {
    cache.delete(key); // Clean up expired entry
    return null;
  }

  return entry.data as T;
}