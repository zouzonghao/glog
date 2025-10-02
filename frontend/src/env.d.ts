/// <reference types="vite/client" />
/// <reference types="astro/client" />

declare namespace App {
    interface Locals {
        isLoggedIn: boolean;
        view: 'list' | 'cards';
        publicSettings: {
            site_title?: string;
            site_description?: string;
            favicon?: string;
            [key: string]: any; // Allow other settings
        };
    }
}

declare namespace svelte.JSX {
    interface HTMLAttributes<T> {
        onintersect?: () => void;
    }
}
/// <reference types="@astrojs/svelte/client" />

declare module '*.svelte' {
    import type { ComponentType } from 'svelte';
    const component: ComponentType;
    export default component;
}

interface ImportMetaEnv {
    readonly PUBLIC_API_URL: string;
}

interface ImportMeta {
    readonly env: ImportMetaEnv;
}