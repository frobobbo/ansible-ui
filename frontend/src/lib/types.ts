export type Role = 'admin' | 'editor' | 'viewer';
export type FieldType = 'text' | 'number' | 'bool' | 'select';
export type RunStatus = 'pending' | 'running' | 'success' | 'failed';

export interface User {
	id: string;
	username: string;
	role: Role;
	created_at: string;
}

export interface Server {
	id: string;
	name: string;
	host: string;
	port: number;
	username: string;
	ssh_private_key?: string;
	pre_command: string;
	created_at: string;
}

export interface Playbook {
	id: string;
	name: string;
	description: string;
	file_path: string;
	created_at: string;
}

export interface FormField {
	id: string;
	form_id: string;
	name: string;
	label: string;
	field_type: FieldType;
	default_value: string;
	options: string; // JSON array string e.g. '["opt1","opt2"]'
	required: boolean;
	sort_order: number;
}

export interface Vault {
	id: string;
	name: string;
	description: string;
	vault_file_name: string; // empty string if no file uploaded
	created_at: string;
}

export interface Form {
	id: string;
	name: string;
	description: string;
	playbook_id: string;
	server_id: string;
	vault_id?: string | null;
	is_quick_action: boolean;
	image_name: string;
	fields?: FormField[];
	created_at: string;
	updated_at: string;
}

export interface Run {
	id: string;
	form_id: string | null;
	playbook_id: string;
	server_id: string;
	variables: string; // JSON string
	status: RunStatus;
	output: string;
	started_at: string | null;
	finished_at: string | null;
}

export interface AuthResponse {
	token: string;
	user: User;
}
