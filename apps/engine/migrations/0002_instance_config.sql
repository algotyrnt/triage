-- Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
-- SPDX-License-Identifier: Apache-2.0

-- Instance-level configuration KV store (GitHub App credentials, OAuth credentials)
CREATE TABLE IF NOT EXISTS instance_config (
  key VARCHAR(128) PRIMARY KEY,
  value TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- GitHub App installations (org → installation mapping)
CREATE TABLE IF NOT EXISTS github_installations (
  id VARCHAR(64) PRIMARY KEY,
  installation_id BIGINT UNIQUE NOT NULL,
  org_login VARCHAR(255) NOT NULL,
  org_id BIGINT,
  account_type VARCHAR(32) DEFAULT 'Organization',
  status VARCHAR(32) DEFAULT 'active',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Repos accessible via a GitHub App installation
CREATE TABLE IF NOT EXISTS installation_repos (
  id VARCHAR(64) PRIMARY KEY,
  installation_id BIGINT NOT NULL REFERENCES github_installations(installation_id),
  owner VARCHAR(255) NOT NULL,
  repo VARCHAR(255) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT unique_install_repo UNIQUE (installation_id, owner, repo)
);
