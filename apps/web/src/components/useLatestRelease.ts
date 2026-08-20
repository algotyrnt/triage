/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import { useState, useEffect } from 'react';

export interface ReleaseInfo {
  version: string;
  releaseUrl: string;
  name: string;
  isLatest: boolean;
}

const DEFAULT_RELEASE: ReleaseInfo = {
  version: 'v0.1.0',
  releaseUrl: 'https://github.com/algotyrnt/triage/releases',
  name: 'v0.1.0',
  isLatest: false,
};

const STORAGE_KEY = 'triage_latest_github_release';
const CACHE_TTL_MS = 10 * 60 * 1000; // 10 minutes cache

export function useLatestRelease(): ReleaseInfo {
  const [release, setRelease] = useState<ReleaseInfo>(() => {
    if (typeof window !== 'undefined') {
      try {
        const cached = localStorage.getItem(STORAGE_KEY);
        if (cached) {
          const parsed = JSON.parse(cached);
          if (parsed.timestamp && Date.now() - parsed.timestamp < CACHE_TTL_MS && parsed.data) {
            return parsed.data;
          }
        }
      } catch {
        // ignore storage errors
      }
    }
    return DEFAULT_RELEASE;
  });

  useEffect(() => {
    let isMounted = true;

    async function fetchRelease() {
      try {
        const res = await fetch('https://api.github.com/repos/algotyrnt/triage/releases/latest');
        if (!res.ok) {
          // Fallback to list releases if /releases/latest is not available
          const fallbackRes = await fetch('https://api.github.com/repos/algotyrnt/triage/releases');
          if (fallbackRes.ok) {
            const releases = await fallbackRes.json();
            if (Array.isArray(releases) && releases.length > 0) {
              const latest = releases[0];
              const data: ReleaseInfo = {
                version: latest.tag_name || 'v0.1.0',
                releaseUrl: latest.html_url || 'https://github.com/algotyrnt/triage/releases',
                name: latest.name || latest.tag_name || 'v0.1.0',
                isLatest: true,
              };
              if (isMounted) setRelease(data);
              try {
                localStorage.setItem(STORAGE_KEY, JSON.stringify({ timestamp: Date.now(), data }));
              } catch {}
            }
          }
          return;
        }

        const latest = await res.json();
        if (latest && latest.tag_name) {
          const data: ReleaseInfo = {
            version: latest.tag_name,
            releaseUrl:
              latest.html_url ||
              `https://github.com/algotyrnt/triage/releases/tag/${latest.tag_name}`,
            name: latest.name || latest.tag_name,
            isLatest: true,
          };
          if (isMounted) setRelease(data);
          try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify({ timestamp: Date.now(), data }));
          } catch {}
        }
      } catch {
        // silently fallback to default
      }
    }

    fetchRelease();

    return () => {
      isMounted = false;
    };
  }, []);

  return release;
}
