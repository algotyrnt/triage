/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import { useState, useEffect } from 'react';
import { engineClient } from '@/services/engineClient';

export interface VersionInfo {
  engineVersion: string | null;
  latestVersion: string | null;
  hasUpdate: boolean;
  releaseUrl: string;
  publishedAt?: string;
}

export function compareVersions(current: string, latest: string): boolean {
  const cleanCurrent = current
    .replace(/^v/, '')
    .split('.')
    .map((n) => parseInt(n, 10) || 0);
  const cleanLatest = latest
    .replace(/^v/, '')
    .split('.')
    .map((n) => parseInt(n, 10) || 0);

  for (let i = 0; i < Math.max(cleanCurrent.length, cleanLatest.length); i++) {
    const c = cleanCurrent[i] || 0;
    const l = cleanLatest[i] || 0;
    if (l > c) return true;
    if (l < c) return false;
  }
  return false;
}

export function useLatestRelease(): VersionInfo {
  const [versionInfo, setVersionInfo] = useState<VersionInfo>({
    engineVersion: null,
    latestVersion: null,
    hasUpdate: false,
    releaseUrl: 'https://github.com/algotyrnt/triage/releases/latest',
  });

  useEffect(() => {
    let cancelled = false;

    async function loadVersionAndCheckRelease() {
      // 1. Fetch live engine version from Triage Engine API
      let resolvedEngineVer = await engineClient.getEngineVersion();
      if (!resolvedEngineVer) {
        resolvedEngineVer = 'v0.1.0';
      }

      if (cancelled) return;

      const formattedEngineVer = resolvedEngineVer.startsWith('v')
        ? resolvedEngineVer
        : `v${resolvedEngineVer}`;

      setVersionInfo((prev) => ({
        ...prev,
        engineVersion: formattedEngineVer,
        releaseUrl: `https://github.com/algotyrnt/triage/releases/tag/${formattedEngineVer}`,
      }));

      // 2. Fetch latest release from GitHub API
      try {
        const res = await fetch('https://api.github.com/repos/algotyrnt/triage/releases/latest', {
          headers: { Accept: 'application/vnd.github.v3+json' },
        });

        if (!res.ok) return;

        const data = await res.json();
        if (cancelled || !data || !data.tag_name) return;

        const latestTag = data.tag_name.startsWith('v') ? data.tag_name : `v${data.tag_name}`;
        const hasUpdate =
          formattedEngineVer !== 'dev' && compareVersions(formattedEngineVer, latestTag);

        setVersionInfo({
          engineVersion: formattedEngineVer,
          latestVersion: latestTag,
          hasUpdate,
          releaseUrl:
            data.html_url || `https://github.com/algotyrnt/triage/releases/tag/${latestTag}`,
          publishedAt: data.published_at,
        });
      } catch {
        // Fallback silently if offline or rate-limited
      }
    }

    loadVersionAndCheckRelease();

    return () => {
      cancelled = true;
    };
  }, []);

  return versionInfo;
}
