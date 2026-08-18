/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

const SENSITIVE_KEYS = new Set([
  'apikey',
  'api_key',
  'token',
  'accesstoken',
  'access_token',
  'authorization',
  'password',
  'secret',
  'clientsecret',
  'client_secret',
  'privatekey',
  'private_key',
  'pem',
  'sessionsecret',
  'session_secret',
]);

const SENSITIVE_STRING_PATTERNS = [
  /tr_[a-zA-Z0-9_-]{8,}/g, // Triage API keys
  /gh[pousr]_[a-zA-Z0-9]{20,}/g, // GitHub tokens
  /eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+/g, // JWT tokens
];

function sanitizeString(str: string): string {
  let sanitized = str;
  for (const pattern of SENSITIVE_STRING_PATTERNS) {
    sanitized = sanitized.replace(pattern, '[REDACTED]');
  }
  return sanitized;
}

function sanitizeData(data: any, seen = new WeakSet()): any {
  if (data === null || data === undefined) return data;

  if (typeof data === 'string') {
    return sanitizeString(data);
  }

  if (typeof data === 'number' || typeof data === 'boolean') {
    return data;
  }

  if (data instanceof Error) {
    return {
      name: data.name,
      message: sanitizeString(data.message),
      stack: data.stack ? sanitizeString(data.stack) : undefined,
    };
  }

  if (typeof data === 'object') {
    if (seen.has(data)) return '[Circular]';
    seen.add(data);

    if (Array.isArray(data)) {
      return data.map((item) => sanitizeData(item, seen));
    }

    const cleaned: Record<string, any> = {};
    for (const [key, value] of Object.entries(data)) {
      const lowerKey = key.toLowerCase().replace(/[^a-z0-9]/g, '');
      if (SENSITIVE_KEYS.has(lowerKey)) {
        cleaned[key] = '[REDACTED]';
      } else {
        cleaned[key] = sanitizeData(value, seen);
      }
    }
    return cleaned;
  }

  return data;
}

class Logger {
  private isProduction = process.env.NODE_ENV === 'production';

  debug(message: string, ...args: any[]) {
    if (this.isProduction) return;
    console.debug(`[TRIAGE:DEBUG] ${message}`, ...args.map((a) => sanitizeData(a)));
  }

  info(message: string, ...args: any[]) {
    console.info(`[TRIAGE:INFO] ${message}`, ...args.map((a) => sanitizeData(a)));
  }

  warn(message: string, ...args: any[]) {
    console.warn(`[TRIAGE:WARN] ${message}`, ...args.map((a) => sanitizeData(a)));
  }

  error(message: string, ...args: any[]) {
    console.error(`[TRIAGE:ERROR] ${message}`, ...args.map((a) => sanitizeData(a)));
  }
}

export const logger = new Logger();
