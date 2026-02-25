<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { forms as formsApi, servers as serversApi, playbooks as playbooksApi, vaults as vaultsApi, ApiError } from '$lib/api';
	import type { Server, Playbook, Vault, FormField, FieldType } from '$lib/types';

	let serverList = $state<Server[]>([]);
	let playbookList = $state<Playbook[]>([]);
	let vaultList = $state<Vault[]>([]);
	let formData = $state({ name: '', description: '', server_id: '', playbook_id: '', vault_id: '' });
	let fields = $state<Partial<FormField>[]>([]);
	let saving = $state(false);
	let error = $state('');

	onMount(async () => {
		[serverList, playbookList, vaultList] = await Promise.all([serversApi.list(), playbooksApi.list(), vaultsApi.list()]);
	});

	function addField() {
		fields = [...fields, { name: '', label: '', field_type: 'text' as FieldType, default_value: '', options: '[]', required: false, sort_order: fields.length }];
	}

	function removeField(i: number) {
		fields = fields.filter((_, idx) => idx !== i);
	}

	function getOptions(f: Partial<FormField>) {
		try { return JSON.parse(f.options || '[]').join(', '); } catch { return ''; }
	}

	function setOptions(f: Partial<FormField>, val: string) {
		f.options = JSON.stringify(val.split(',').map(s => s.trim()).filter(Boolean));
	}

	async function save() {
		saving = true;
		error = '';
		try {
			await formsApi.create({ ...formData, fields });
			goto('/forms');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}
</script>

<div class="page-header">
	<h1>New Form</h1>
	<a href="/forms" class="btn btn-secondary">← Back</a>
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

<form onsubmit={(e) => { e.preventDefault(); save(); }}>
	<div class="card">
		<h2>Basic Info</h2>
		<div class="grid-2">
			<div class="form-group">
				<label>Form Name</label>
				<input class="form-control" bind:value={formData.name} required />
			</div>
			<div class="form-group">
				<label>Description</label>
				<input class="form-control" bind:value={formData.description} />
			</div>
			<div class="form-group">
				<label>Server</label>
				<select class="form-control" bind:value={formData.server_id} required>
					<option value="">Select server...</option>
					{#each serverList as sv}<option value={sv.id}>{sv.name} ({sv.host})</option>{/each}
				</select>
			</div>
			<div class="form-group">
				<label>Playbook</label>
				<select class="form-control" bind:value={formData.playbook_id} required>
					<option value="">Select playbook...</option>
					{#each playbookList as pb}<option value={pb.id}>{pb.name}</option>{/each}
				</select>
			</div>
			<div class="form-group">
				<label>Vault (optional)</label>
				<select class="form-control" bind:value={formData.vault_id}>
					<option value="">None</option>
					{#each vaultList as v}<option value={v.id}>{v.name}</option>{/each}
				</select>
				<small class="hint">Select a vault to pass --vault-password-file when running this form.</small>
			</div>
		</div>
	</div>

	<div class="card">
		<div class="section-header">
			<h2>Fields</h2>
			<button type="button" class="btn btn-secondary btn-sm" onclick={addField}>+ Add Field</button>
		</div>
		{#if fields.length === 0}
			<p class="empty-state" style="padding:1rem 0">No fields yet. Add fields to capture Ansible variables.</p>
		{/if}
		{#each fields as field, i}
			<div class="field-row">
				<div class="field-grid">
					<div class="form-group">
						<label>Variable Name</label>
						<input class="form-control" bind:value={field.name} placeholder="e.g. db_host" required />
					</div>
					<div class="form-group">
						<label>Label</label>
						<input class="form-control" bind:value={field.label} placeholder="Display label" required />
					</div>
					<div class="form-group">
						<label>Type</label>
						<select class="form-control" bind:value={field.field_type}>
							<option value="text">Text</option>
							<option value="number">Number</option>
							<option value="bool">Boolean</option>
							<option value="select">Select</option>
						</select>
					</div>
					<div class="form-group">
						<label>Default Value</label>
						<input class="form-control" bind:value={field.default_value} />
					</div>
					{#if field.field_type === 'select'}
						<div class="form-group" style="grid-column: span 2">
							<label>Options (comma-separated)</label>
							<input class="form-control" value={getOptions(field)} oninput={(e) => setOptions(field, (e.target as HTMLInputElement).value)} placeholder="opt1, opt2, opt3" />
						</div>
					{/if}
					<div class="form-group field-required">
						<label><input type="checkbox" bind:checked={field.required} /> Required</label>
					</div>
				</div>
				<button type="button" class="btn btn-sm btn-danger field-remove" onclick={() => removeField(i)}>✕</button>
			</div>
		{/each}
	</div>

	<div class="actions" style="justify-content:flex-end">
		<a href="/forms" class="btn btn-secondary">Cancel</a>
		<button type="submit" class="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Create Form'}</button>
	</div>
</form>

<style>
	.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
	.section-header h2 { margin-bottom: 0; }
	.field-row { display: flex; gap: 0.75rem; align-items: flex-start; padding: 1rem; border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 0.75rem; }
	.field-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0.75rem; flex: 1; }
	.field-required { display: flex; align-items: center; }
	.field-required label { display: flex; align-items: center; gap: 0.375rem; font-weight: normal; margin-bottom: 0; }
	.field-remove { align-self: flex-end; margin-bottom: 1rem; }
</style>
