declare global {
	namespace App {}

	interface ImportMetaEnv {
		readonly VITE_CONTEXTOS_DEFAULT_WORKSPACE?: string;
		readonly VITE_CONTEXTOS_DEBUG_LOGS?: string;
	}

	interface ImportMeta {
		readonly env: ImportMetaEnv;
	}
}

export {};
