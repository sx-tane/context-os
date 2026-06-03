declare global {
	namespace App {}

	interface ImportMetaEnv {
		readonly VITE_CONTEXTOS_DEFAULT_WORKSPACE?: string;
	}

	interface ImportMeta {
		readonly env: ImportMetaEnv;
	}
}

export {};
