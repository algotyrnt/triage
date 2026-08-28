/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import pkg from '../../package.json';

export interface ReleaseInfo {
  version: string;
  releaseUrl: string;
  name: string;
}

// Injected at build time via PUBLIC_TRIAGE_VERSION environment variable or resolved from package.json
const rawVersion: string =
  (typeof import.meta !== 'undefined' && import.meta.env?.PUBLIC_TRIAGE_VERSION) ||
  pkg.version ||
  '0.1.0';

const formattedVersion = rawVersion.startsWith('v') ? rawVersion : `v${rawVersion}`;

export const CURRENT_RELEASE: ReleaseInfo = {
  version: formattedVersion,
  releaseUrl: `https://github.com/algotyrnt/triage/releases/tag/${formattedVersion}`,
  name: formattedVersion,
};

/**
 * Returns the release version metadata baked in at build/release time.
 * Eliminates client-side runtime GitHub API fetching and rate limits.
 */
export function useLatestRelease(): ReleaseInfo {
  return CURRENT_RELEASE;
}
