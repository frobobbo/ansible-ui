<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { forms as formsApi, servers as serversApi, playbooks as playbooksApi, vaults as vaultsApi, ApiError } from '$lib/api';
	import type { Server, Playbook, Vault, FormField, FieldType } from '$lib/types';

	let id = $derived($page.params.id);
	let serverList = $state<Server[]>([]);
	let playbookList = $state<Playbook[]>([]);
	let vaultList = $state<Vault[]>([]);
	let formData = $state({ name: '', description: '', server_id: '', playbook_id: '', vault_id: '', is_quick_action: false, schedule_cron: '', schedule_enabled: false });
	let nextRunAt = $state<string | null>(null);
	let imageName = $state('');
	let imageUploading = $state(false);
	let fields = $state<Partial<FormField>[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');

	onMount(async () => {
		const [form, svList, pbList, vList] = await Promise.all([
			formsApi.get(id),
			serversApi.list(),
			playbooksApi.list(),
			vaultsApi.list()
		]);
		serverList = svList;
		playbookList = pbList;
		vaultList = vList;
		if (form) {
			formData = { name: form.name, description: form.description, server_id: form.server_id, playbook_id: form.playbook_id, vault_id: form.vault_id ?? '', is_quick_action: form.is_quick_action, schedule_cron: form.schedule_cron ?? '', schedule_enabled: form.schedule_enabled ?? false };
			nextRunAt = form.next_run_at ?? null;
			imageName = form.image_name;
			fields = form.fields || [];
		}
		loading = false;
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
			await formsApi.update(id, { ...formData, fields });
			goto('/forms');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	async function uploadImage(input: HTMLInputElement) {
		const file = input.files?.[0];
		if (!file) return;
		imageUploading = true;
		try {
			const updated = await formsApi.uploadImage(id, file);
			imageName = updated.image_name;
		} catch (err) {
			alert(err instanceof ApiError ? err.message : 'Upload failed');
		} finally {
			imageUploading = false;
			input.value = '';
		}
	}

	async function removeImage() {
		if (!confirm('Remove the image from this form?')) return;
		try {
			const updated = await formsApi.deleteImage(id);
			imageName = updated.image_name;
		} catch {
			alert('Failed to remove image');
		}
	}
</script>

<div class="page-header">
	<h1>Edit Form</h1>
	<div class="actions">
		<a href="/forms/{id}/run" class="btn btn-primary">Run</a>
		<a href="/forms" class="btn btn-secondary">← Back</a>
	</div>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else}
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
				<div class="form-group">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={formData.is_quick_action} />
						Show as Quick Action on dashboard
					</label>
					<small class="hint">Quick actions appear as clickable cards on the dashboard for all users.</small>
				</div>
			</div>
			<div class="form-group" style="margin-top:0.5rem">
				<label>Quick Action Image (optional)</label>
				{#if imageName}
					<div class="image-preview-row">
						<img src="/api/forms/{id}/image" alt={formData.name} class="image-preview" />
						<div>
							<div class="file-badge">{imageName}</div>
							<button type="button" class="btn btn-sm btn-danger" style="margin-top:0.5rem" onclick={removeImage}>Remove Image</button>
						</div>
					</div>
				{:else}
					<label class="btn btn-secondary file-label" class:disabled={imageUploading}>
						{imageUploading ? 'Uploading…' : 'Choose Image…'}
						<input type="file" accept="image/*" style="display:none" disabled={imageUploading}
							onchange={(e) => uploadImage(e.currentTarget as HTMLInputElement)} />
					</label>
				{/if}
				<small class="hint">Displayed on the quick action card. PNG, JPG, SVG, etc.</small>
			</div>
		</div>

		<div class="card">
			<div class="section-header">
				<h2>Fields</h2>
				<button type="button" class="btn btn-secondary btn-sm" onclick={addField}>+ Add Field</button>
			</div>
			{#if fields.length === 0}
				<p class="empty-state" style="padding:1rem 0">No fields. Add fields to capture Ansible variables.</p>
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
							<input class="form-control" bind:value={field.label} required />
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
							<label>Default</label>
							<input class="form-control" bind:value={field.default_value} />
						</div>
						{#if field.field_type === 'select'}
							<div class="form-group" style="grid-column: span 2">
								<label>Options (comma-separated)</label>
								<input class="form-control" value={getOptions(field)} oninput={(e) => setOptions(field, (e.target as HTMLInputElement).value)} />
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

		<div class="card">
			<h2>Scheduling</h2>
			<div class="form-group">
				<label class="checkbox-label">
					<input type="checkbox" bind:checked={formData.schedule_enabled} />
					Run on a schedule
				</label>
				<small class="hint">Runs automatically using field default values. Times are UTC.</small>
			</div>
			{#if formData.schedule_enabled}
				<div class="form-group">
					<label for="sched_cron">Cron Expression</label>
					<input id="sched_cron" class="form-control" bind:value={formData.schedule_cron}
						placeholder="0 2 * * *" required={formData.schedule_enabled} />
					<small class="hint">5-field cron (min hr dom mon dow) or @hourly · @daily · @weekly</small>
					{#if nextRunAt}
						<small class="hint">Next run: {new Date(nextRunAt).toLocaleString()} UTC</small>
					{/if}
				</div>
			{/if}
		</div>

		<div class="actions" style="justify-content:flex-end">
			<a href="/forms" class="btn btn-secondary">Cancel</a>
			<button type="submit" class="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save Changes'}</button>
		</div>
	</form>
{/if}

<style>
	.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
	.section-header h2 { margin-bottom: 0; }
	.field-row { display: flex; gap: 0.75rem; align-items: flex-start; padding: 1rem; border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 0.75rem; }
	.field-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0.75rem; flex: 1; }
	.field-required { display: flex; align-items: center; }
	.field-required label { display: flex; align-items: center; gap: 0.375rem; font-weight: normal; margin-bottom: 0; }
	.field-remove { align-self: flex-end; margin-bottom: 1rem; }
	.checkbox-label { display: flex; align-items: center; gap: 0.5rem; font-weight: 500; cursor: pointer; }
	.image-preview-row { display: flex; align-items: flex-start; gap: 1rem; margin-bottom: 0.5rem; }
	.image-preview { width: 80px; height: 80px; object-fit: cover; border-radius: var(--radius); border: 1px solid var(--border); }
	.file-badge { display: inline-flex; align-items: center; background: #f0fdf4; border: 1px solid #bbf7d0; color: #166534; border-radius: 4px; padding: 0.15rem 0.5rem; font-size: 0.8rem; }
	.file-label { cursor: pointer; display: inline-flex; align-items: center; }
	.file-label.disabled { opacity: 0.6; cursor: not-allowed; }
	.hint { display: block; margin-top: 0.25rem; font-size: 0.8rem; color: #64748b; }
</style>
