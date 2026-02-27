<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/stores';
	import { forms as formsApi, runs as runsApi, ApiError } from '$lib/api';
	import { isEditor, authStore } from '$lib/stores';
	import type { Form, Run } from '$lib/types';
	import AnsiToHtml from 'ansi-to-html';

	const conv = new AnsiToHtml({ escapeXML: true });

	let id = $derived($page.params.id);
	let form = $state<Form | null>(null);
	let loading = $state(true);
	let variables = $state<Record<string, string>>({});
	let running = $state(false);
	let runResult = $state<Run | null>(null);
	let error = $state('');
	let currentRunId = $state<string | null>(null);
	let outputLines = $state<string[]>([]);
	let es: EventSource | null = null;

	onMount(async () => {
		form = await formsApi.get(id);
		if (form?.fields) {
			for (const f of form.fields) {
				variables[f.name] = f.default_value || (f.field_type === 'bool' ? 'false' : '');
			}
		}
		loading = false;
	});

	onDestroy(() => { es?.close(); });

	async function executeRun() {
		if (!form) return;
		running = true;
		error = '';
		runResult = null;
		outputLines = [];
		currentRunId = null;

		const typedVars: Record<string, unknown> = {};
		for (const field of form.fields || []) {
			const val = variables[field.name];
			if (field.field_type === 'number') typedVars[field.name] = Number(val);
			else if (field.field_type === 'bool') typedVars[field.name] = val === 'true';
			else typedVars[field.name] = val;
		}

		try {
			const { run_id } = await runsApi.create(id, typedVars);
			currentRunId = run_id;

			const token = encodeURIComponent(get(authStore).token ?? '');
			es = new EventSource(`/api/runs/${run_id}/stream?token=${token}`);

			es.onmessage = (e) => {
				outputLines = [...outputLines, e.data];
			};

			es.addEventListener('done', () => {
				es?.close(); es = null; running = false;
				runsApi.get(run_id).then((r) => { if (r) runResult = r; });
			});

			es.onerror = () => {
				es?.close(); es = null; running = false;
				runsApi.get(run_id).then((r) => { if (r) runResult = r; });
			};
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Failed to start run';
			running = false;
		}
	}

	async function cancelRun() {
		if (!currentRunId) return;
		try { await runsApi.cancel(currentRunId); } catch { /* ignore */ }
	}

	function statusClass(status: string) {
		return { pending: 'badge-muted', running: 'badge-info', success: 'badge-success', failed: 'badge-danger' }[status] || 'badge-muted';
	}

	let liveOutputHtml = $derived(conv.toHtml(outputLines.join('\n')));
	let finalOutputHtml = $derived(conv.toHtml(runResult?.output || ''));
</script>

<div class="page-header">
	<h1>Run: {form?.name ?? '...'}</h1>
	{#if $isEditor}
		<a href="/forms/{id}" class="btn btn-secondary">← Edit Form</a>
	{/if}
</div>

{#if loading}
	<p class="empty-state">Loading form...</p>
{:else if form}
	{#if error}<div class="alert alert-error">{error}</div>{/if}

	<form onsubmit={(e) => { e.preventDefault(); executeRun(); }}>
		<div class="card">
			<h2>Variables</h2>
			{#if !form.fields || form.fields.length === 0}
				<p class="empty-state" style="padding:0.5rem 0">This form has no fields. The playbook will run with no extra variables.</p>
			{/if}
			{#each form.fields ?? [] as field}
				<div class="form-group">
					<label>
						{field.label}
						{#if field.required}<span class="required">*</span>{/if}
						<span class="var-name">({field.name})</span>
					</label>
					{#if field.field_type === 'bool'}
						<select class="form-control" bind:value={variables[field.name]}>
							<option value="false">false</option>
							<option value="true">true</option>
						</select>
					{:else if field.field_type === 'select'}
						<select class="form-control" bind:value={variables[field.name]}>
							{#each JSON.parse(field.options || '[]') as opt}
								<option value={opt}>{opt}</option>
							{/each}
						</select>
					{:else if field.field_type === 'number'}
						<input class="form-control" type="number" bind:value={variables[field.name]} required={field.required} />
					{:else}
						<input class="form-control" type="text" bind:value={variables[field.name]} required={field.required} />
					{/if}
				</div>
			{/each}
		</div>

		<div class="actions" style="justify-content:flex-end; margin-bottom:1.5rem">
			{#if running && currentRunId}
				<button type="button" class="btn btn-danger" onclick={cancelRun}>Cancel</button>
			{/if}
			<button type="submit" class="btn btn-primary" disabled={running}>
				{running ? 'Running...' : '▶ Run Playbook'}
			</button>
		</div>
	</form>

	{#if running || runResult}
		<div class="card">
			<div class="run-header">
				<h2>Run Output</h2>
				<div class="run-meta">
					{#if runResult}
						<span class="badge {statusClass(runResult.status)}">{runResult.status}</span>
						{#if runResult.started_at}
							<span class="meta-text">Started: {new Date(runResult.started_at).toLocaleString()}</span>
						{/if}
						{#if runResult.finished_at}
							<span class="meta-text">Finished: {new Date(runResult.finished_at).toLocaleString()}</span>
						{/if}
						<a href="/runs/{runResult.id}" class="btn btn-sm btn-secondary">View Full Run</a>
					{:else}
						<span class="badge badge-info">running</span>
						<span class="streaming">● Live</span>
					{/if}
				</div>
			</div>
			<pre class="output">{@html running
				? (liveOutputHtml || '<span class="muted-out">Waiting for output…</span>')
				: (finalOutputHtml || '<span class="muted-out">No output.</span>')}</pre>
		</div>
	{/if}
{/if}

<style>
	.required { color: var(--danger); margin-left: 2px; }
	.var-name { font-size: 0.75rem; color: var(--text-muted); font-weight: normal; font-family: monospace; }
	.run-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.5rem; }
	.run-header h2 { margin-bottom: 0; }
	.run-meta { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
	.meta-text { font-size: 0.8rem; color: var(--text-muted); }
	.streaming { font-size: 0.8rem; color: var(--primary); font-weight: 600; }
	.output { background: #0f172a; color: #e2e8f0; padding: 1.25rem; border-radius: var(--radius); font-size: 0.8rem; line-height: 1.6; overflow-x: auto; white-space: pre-wrap; word-break: break-all; max-height: 500px; overflow-y: auto; }
	:global(.muted-out) { color: #64748b; font-style: italic; }
</style>
