import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'info';

export interface Toast {
	id: number;
	message: string;
	type: ToastType;
}

function createToastStore() {
	const { subscribe, update } = writable<Toast[]>([]);
	let nextId = 0;

	function add(message: string, type: ToastType, duration: number) {
		const id = ++nextId;
		update((toasts) => [...toasts, { id, message, type }]);
		setTimeout(() => remove(id), duration);
	}

	function remove(id: number) {
		update((toasts) => toasts.filter((t) => t.id !== id));
	}

	return {
		subscribe,
		success: (msg: string) => add(msg, 'success', 4000),
		error: (msg: string) => add(msg, 'error', 6000),
		info: (msg: string) => add(msg, 'info', 4000),
		remove
	};
}

export const toast = createToastStore();

// ── Confirm dialog ────────────────────────────────────────────────────────────

interface ConfirmState {
	message: string;
	confirmText: string;
	danger: boolean;
	resolve: (value: boolean) => void;
}

export const confirmState = writable<ConfirmState | null>(null);

export function confirmDialog(
	message: string,
	opts: { confirmText?: string; danger?: boolean } = {}
): Promise<boolean> {
	return new Promise((resolve) => {
		confirmState.set({
			message,
			confirmText: opts.confirmText ?? 'Delete',
			danger: opts.danger ?? true,
			resolve
		});
	});
}
