// Tiny toast store (Svelte 5 runes). Used for feedback after enhanced form submissions.
export type ToastTone = 'success' | 'error' | 'info';

interface Toast {
	id: number;
	tone: ToastTone;
	message: string;
}

let next = 1;
export const toasts = $state<Toast[]>([]);

export function toast(message: string, tone: ToastTone = 'success', ms = 4000) {
	const id = next++;
	toasts.push({ id, tone, message });
	setTimeout(() => dismiss(id), ms);
}

export function dismiss(id: number) {
	const i = toasts.findIndex((t) => t.id === id);
	if (i >= 0) toasts.splice(i, 1);
}
