import type {
  BorgSTMFConfig,
  FormData,
  FormField,
  EncryptResult,
  KeyPair,
  InterceptorOptions,
  BorgSTMFWasm,
} from './types';

export * from './types';

const DEFAULT_FIELD_NAME = '_stmf_payload';
const DEFAULT_WASM_PATH = './stmf.wasm';

/**
 * BorgSTMF - Sovereign Form Encryption
 *
 * Encrypts HTML form data client-side using the server's public key.
 * Data is encrypted with X25519 ECDH + ChaCha20-Poly1305, providing
 * end-to-end encryption even against MITM proxies.
 *
 * @example
 * ```typescript
 * const borg = new BorgSTMF({
 *   serverPublicKey: 'base64PublicKeyHere',
 *   wasmPath: '/wasm/stmf.wasm'
 * });
 *
 * await borg.init();
 *
 * // Manual encryption
 * const result = await borg.encryptForm(document.querySelector('form'));
 *
 * // Or use interceptor
 * borg.enableInterceptor();
 * ```
 */
export class BorgSTMF {
  private config: Required<BorgSTMFConfig>;
  private wasm: BorgSTMFWasm | null = null;
  private initialized = false;
  private interceptorActive = false;
  private interceptorHandler: ((e: Event) => void) | null = null;

  constructor(config: BorgSTMFConfig) {
    this.config = {
      serverPublicKey: config.serverPublicKey,
      wasmPath: config.wasmPath || DEFAULT_WASM_PATH,
      fieldName: config.fieldName || DEFAULT_FIELD_NAME,
      debug: config.debug || false,
    };
  }

  /**
   * Initialize the WASM module. Must be called before encryption.
   */
  async init(): Promise<void> {
    if (this.initialized) return;

    // Check if WASM is already loaded (e.g., from a script tag)
    if (window.BorgSTMF?.ready) {
      this.wasm = window.BorgSTMF;
      this.initialized = true;
      this.log('Using pre-loaded WASM module');
      return;
    }

    // Load wasm_exec.js if not already loaded
    if (typeof Go === 'undefined') {
      await this.loadScript(this.config.wasmPath.replace('stmf.wasm', 'wasm_exec.js'));
    }

    // Load and instantiate the WASM module
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(
      fetch(this.config.wasmPath),
      go.importObject
    );

    // Run the Go main function
    go.run(result.instance);

    // Wait for WASM to be ready
    await this.waitForWasm();

    this.wasm = window.BorgSTMF!;
    this.initialized = true;
    this.log('WASM module initialized, version:', this.wasm.version);
  }

  /**
   * Encrypt an HTML form element
   */
  async encryptForm(form: HTMLFormElement): Promise<EncryptResult> {
    this.ensureInitialized();

    const formData = new window.FormData(form);
    return this.encryptFormData(formData);
  }

  /**
   * Encrypt a FormData object
   */
  async encryptFormData(formData: globalThis.FormData): Promise<EncryptResult> {
    this.ensureInitialized();

    const fields: Record<string, string | FormField> = {};

    formData.forEach((value, key) => {
      if (value instanceof File) {
        // Handle file uploads - read as base64
        // Note: For large files, consider chunking or streaming
        this.log('File field detected:', key, value.name);
        // For now, skip files - they need async reading
        // TODO: Add file support with FileReader
      } else {
        fields[key] = value.toString();
      }
    });

    const payload = await this.wasm!.encryptFields(
      fields,
      this.config.serverPublicKey,
      {
        origin: window.location.origin,
        timestamp: Date.now().toString(),
      }
    );

    return {
      payload,
      fieldName: this.config.fieldName,
    };
  }

  /**
   * Encrypt a simple key-value object
   */
  async encryptFields(
    fields: Record<string, string>,
    metadata?: Record<string, string>
  ): Promise<EncryptResult> {
    this.ensureInitialized();

    const meta = {
      origin: window.location.origin,
      timestamp: Date.now().toString(),
      ...metadata,
    };

    const payload = await this.wasm!.encryptFields(
      fields,
      this.config.serverPublicKey,
      meta
    );

    return {
      payload,
      fieldName: this.config.fieldName,
    };
  }

  /**
   * Encrypt a full FormData structure
   */
  async encryptFormDataStruct(data: FormData): Promise<EncryptResult> {
    this.ensureInitialized();

    const payload = await this.wasm!.encrypt(
      JSON.stringify(data),
      this.config.serverPublicKey
    );

    return {
      payload,
      fieldName: this.config.fieldName,
    };
  }

  /**
   * Generate a new keypair (for testing/development only)
   */
  async generateKeyPair(): Promise<KeyPair> {
    this.ensureInitialized();
    return this.wasm!.generateKeyPair();
  }

  /**
   * Enable automatic form interception.
   * Intercepts submit events on forms with the data-stmf attribute.
   */
  enableInterceptor(options: InterceptorOptions = {}): void {
    if (this.interceptorActive) return;

    const { autoSubmit = true } = options;

    this.interceptorHandler = async (e: Event) => {
      const form = e.target as HTMLFormElement;

      // Check if this form should be intercepted
      const publicKey = form.dataset.stmf;
      if (!publicKey && !options.selector) return;
      if (options.selector && !form.matches(options.selector)) return;

      e.preventDefault();
      e.stopPropagation();

      try {
        // Use form's public key or default config
        const serverKey = publicKey || this.config.serverPublicKey;

        // Callback before encryption
        if (options.onBeforeEncrypt) {
          const proceed = await options.onBeforeEncrypt(form);
          if (proceed === false) return;
        }

        // Encrypt the form
        const originalFormData = new window.FormData(form);
        const fields: Record<string, string> = {};

        originalFormData.forEach((value, key) => {
          if (!(value instanceof File)) {
            fields[key] = value.toString();
          }
        });

        const payload = await this.wasm!.encryptFields(
          fields,
          serverKey,
          {
            origin: window.location.origin,
            timestamp: Date.now().toString(),
            formId: form.id || undefined,
          }
        );

        // Callback after encryption
        if (options.onAfterEncrypt) {
          options.onAfterEncrypt(form, payload);
        }

        if (autoSubmit) {
          // Create new form data with only the encrypted payload
          const encryptedFormData = new window.FormData();
          encryptedFormData.append(this.config.fieldName, payload);

          // Submit via fetch
          const response = await fetch(form.action || window.location.href, {
            method: form.method || 'POST',
            body: encryptedFormData,
          });

          // Handle response - trigger custom event
          const event = new CustomEvent('borgstmf:submitted', {
            detail: { form, response, payload },
          });
          form.dispatchEvent(event);
        }
      } catch (error) {
        this.log('Encryption error:', error);
        if (options.onError) {
          options.onError(form, error as Error);
        } else {
          throw error;
        }
      }
    };

    document.addEventListener('submit', this.interceptorHandler, true);
    this.interceptorActive = true;
    this.log('Form interceptor enabled');
  }

  /**
   * Disable automatic form interception
   */
  disableInterceptor(): void {
    if (!this.interceptorActive || !this.interceptorHandler) return;

    document.removeEventListener('submit', this.interceptorHandler, true);
    this.interceptorHandler = null;
    this.interceptorActive = false;
    this.log('Form interceptor disabled');
  }

  /**
   * Check if the module is initialized
   */
  isInitialized(): boolean {
    return this.initialized;
  }

  /**
   * Get the WASM module version
   */
  getVersion(): string {
    return this.wasm?.version || 'not loaded';
  }

  private ensureInitialized(): void {
    if (!this.initialized || !this.wasm) {
      throw new Error('BorgSTMF not initialized. Call init() first.');
    }
  }

  private async waitForWasm(timeout = 5000): Promise<void> {
    const start = Date.now();
    while (!window.BorgSTMF?.ready) {
      if (Date.now() - start > timeout) {
        throw new Error('Timeout waiting for WASM module to initialize');
      }
      await new Promise((resolve) => setTimeout(resolve, 50));
    }
  }

  private async loadScript(src: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = src;
      script.onload = () => resolve();
      script.onerror = () => reject(new Error(`Failed to load ${src}`));
      document.head.appendChild(script);
    });
  }

  private log(...args: unknown[]): void {
    if (this.config.debug) {
      console.log('[BorgSTMF]', ...args);
    }
  }
}

// Export a factory function for convenience
export function createBorgSTMF(config: BorgSTMFConfig): BorgSTMF {
  return new BorgSTMF(config);
}

// Export types for the Go interface
declare class Go {
  constructor();
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
}
