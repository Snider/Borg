/**
 * Configuration options for BorgSTMF
 */
export interface BorgSTMFConfig {
  /**
   * Base64-encoded X25519 public key of the server.
   * Form data will be encrypted using this key.
   */
  serverPublicKey: string;

  /**
   * Path to the WASM file.
   * @default './stmf.wasm'
   */
  wasmPath?: string;

  /**
   * Name of the form field that will contain the encrypted payload.
   * @default '_stmf_payload'
   */
  fieldName?: string;

  /**
   * Enable debug logging.
   * @default false
   */
  debug?: boolean;
}

/**
 * Form field definition
 */
export interface FormField {
  name: string;
  value: string;
  type?: string;
  filename?: string;
  mime?: string;
}

/**
 * Form data structure for encryption
 */
export interface FormData {
  fields: FormField[];
  meta?: Record<string, string>;
}

/**
 * Result of encrypting form data
 */
export interface EncryptResult {
  /** Base64-encoded encrypted STMF payload */
  payload: string;
  /** Name of the form field for the payload */
  fieldName: string;
}

/**
 * X25519 keypair (for testing/development)
 */
export interface KeyPair {
  /** Base64-encoded public key */
  publicKey: string;
  /** Base64-encoded private key (keep secret!) */
  privateKey: string;
}

/**
 * Options for the form interceptor
 */
export interface InterceptorOptions {
  /**
   * CSS selector for forms to intercept.
   * If not specified, intercepts forms with data-stmf attribute.
   */
  selector?: string;

  /**
   * Callback before encryption.
   * Return false to cancel encryption.
   */
  onBeforeEncrypt?: (form: HTMLFormElement) => boolean | Promise<boolean>;

  /**
   * Callback after encryption.
   */
  onAfterEncrypt?: (form: HTMLFormElement, payload: string) => void;

  /**
   * Callback on encryption error.
   */
  onError?: (form: HTMLFormElement, error: Error) => void;

  /**
   * Whether to submit the form automatically after encryption.
   * @default true
   */
  autoSubmit?: boolean;
}

/**
 * BorgSTMF WASM module interface
 */
export interface BorgSTMFWasm {
  encrypt: (formDataJSON: string, serverPublicKey: string) => Promise<string>;
  encryptFields: (
    fields: Record<string, string | FormField>,
    serverPublicKey: string,
    metadata?: Record<string, string>
  ) => Promise<string>;
  generateKeyPair: () => Promise<KeyPair>;
  version: string;
  ready: boolean;
}

declare global {
  interface Window {
    BorgSTMF?: BorgSTMFWasm;
  }
}
