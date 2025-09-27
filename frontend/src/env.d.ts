/// <reference types="vite/client" />
/// <reference types="astro/client" />
/// <reference types="@astrojs/svelte/client" />

declare namespace App {
    interface Locals {
        isLoggedIn: boolean;
    }
}

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