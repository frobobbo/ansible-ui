<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/stores';
	import { runs as runsApi } from '$lib/api';
	import { authStore } from '$lib/stores';
	import type { Run } from '$lib/types';

	let id = $derived($page.params.id);
	let run = $state<Run | null>(null);
	let loading = $state(true);
	let streaming = $state(false);
	let es: EventSource | null = null;

	onMount(async () => {
		run = await runsApi.get(id);
		loading = false;
		if (run && (run.status === 'pending' || run.status === 'running')) {
			startStream();
		}
	});

	onDestroy(() => { es?.close(); });

	function startStream() {
		streaming = true;
		// Clear output so we rebuild it from SSE (output is empty while running)
		if (run) run = { ...run, output: '' };

		const token = get(authStore).token ?? '';
		es = new EventSource(`/api/runs/${id}/stream?token=${encodeURIComponent(token)}`);

		es.onmessage = (e) => {
			if (run) {
				run = { ...run, output: (run.output ?? '') + e.data + '\n' };
			}
		};

		es.addEventListener('done', (e: MessageEvent) => {
			es?.close();
			es = null;
			streaming = false;
			// Refresh to get final DB state (status, finished_at, etc.)
			runsApi.get(id).then((r) => { if (r) run = r; });
		});

		es.onerror = () => {
			es?.close();
			es = null;
			streaming = false;
			// Fall back to one final fetch
			runsApi.get(id).then((r) => { if (r) run = r; });
		};
	}

	function statusClass(status: string) {
		return { pending: 'badge-muted', running: 'badge-info', success: 'badge-success', failed: 'badge-danger' }[status] || 'badge-muted';
	}

	let parsedVars = $derived(() => {
		if (!run?.variables) return {};
		try { return JSON.parse(run.variables); } catch { return {}; }
	});
</script>

<div class="page-header">
	<h1>Run Detail</h1>
	<a href="/runs" class="btn btn-secondary">← Back</a>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if !run}
	<div class="alert alert-error">Run not found.</div>
{:else}
	<div class="card">
		<div class="meta-grid">
			<div><span class="meta-label">Run ID</span><code>{run.id}</code></div>
			<div>
				<span class="meta-label">Status</span>
				<span class="badge {statusClass(run.status)}">{run.status}</span>
				{#if streaming}<span class="streaming">● Live</span>{/if}
			</div>
			<div><span class="meta-label">Started</span>{run.started_at ? new Date(run.started_at).toLocaleString() : '—'}</div>
			<div><span class="meta-label">Finished</span>{run.finished_at ? new Date(run.finished_at).toLocaleString() : '—'}</div>
		</div>

		{#if Object.keys(parsedVars()).length > 0}
			<div style="margin-top:1rem">
				<h3>Variables</h3>
				<table class="table" style="margin-top:0.5rem">
					<thead><tr><th>Variable</th><th>Value</th></tr></thead>
					<tbody>
						{#each Object.entries(parsedVars()) as [k, v]}
							<tr><td><code>{k}</code></td><td>{JSON.stringify(v)}</td></tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>

	<div class="card">
		<div class="output-header">
			<h2>Output</h2>
			{#if streaming}<span class="streaming">● Streaming live output…</span>{/if}
		</div>
		<pre class="output">{run.output || (streaming ? 'Waiting for output…' : 'No output.')}</pre>
	</div>
{/if}

<style>
	.meta-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 1rem; }
	.meta-label { display: block; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); margin-bottom: 0.25rem; }
	.streaming { font-size: 0.8rem; color: var(--primary); font-weight: 600; margin-left: 0.5rem; }
	.output-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 0.75rem; }
	.output-header h2 { margin-bottom: 0; }
	.output { background: #0f172a; color: #e2e8f0; padding: 1.25rem; border-radius: var(--radius); font-size: 0.8rem; line-height: 1.6; overflow-x: auto; white-space: pre-wrap; word-break: break-all; max-height: 600px; overflow-y: auto; }
</style>
