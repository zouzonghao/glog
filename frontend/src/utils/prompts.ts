import { mount } from 'svelte';
import InputPrompt from '../components/InputPrompt.svelte';
import ConfirmPrompt from '../components/ConfirmPrompt.svelte';

interface PromptOptions {
  title: string;
  inputType?: 'text' | 'password' | 'number' | 'email';
}

/**
 * Displays a confirmation modal and returns a promise that resolves with a boolean.
 * @param title - The confirmation message to display.
 * @returns A promise that resolves with true if confirmed, or false if cancelled.
 */
export function showConfirmPrompt(title: string): Promise<boolean> {
  return new Promise((resolve) => {
    const component = mount(ConfirmPrompt, {
      target: document.body,
      props: {
        title,
        onConfirm: () => {
          resolve(true);
        },
        onCancel: () => {
          resolve(false);
        },
      },
    });
  });
}

/**
 * Displays a prompt modal and returns a promise that resolves with the entered value.
 * @param options - The options for the prompt.
 * @returns A promise that resolves with the entered string, or null if the user cancels.
 */
export function showInputPrompt({ title, inputType = 'text' }: PromptOptions): Promise<string | null> {
  return new Promise((resolve) => {
    const component = mount(InputPrompt, {
      target: document.body,
      props: {
        title,
        inputType,
        onConfirm: (value: string) => {
          resolve(value);
        },
        onCancel: () => {
          resolve(null);
        },
      },
    });

    // The component will destroy itself after closing.
    // We don't need to manually call unmount here as the component handles its lifecycle.
  });
}