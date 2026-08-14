-- Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
-- SPDX-License-Identifier: Apache-2.0
--
-- Triage PostgreSQL Schema
-- Full DDL for the complete database. Run once on a fresh instance.

-- ---------------------------------------------------------------------------
-- 1. Users & OAuth Sessions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
  id          VARCHAR(64)  PRIMARY KEY,
  github_id   VARCHAR(64)  UNIQUE NOT NULL,
  username    VARCHAR(255) NOT NULL,
  avatar_url  TEXT,
  role        VARCHAR(32)  DEFAULT 'Member',
  created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------------------------
-- 2. Tracked Repositories & GitHub App Installations
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS repositories (
  id              VARCHAR(64) PRIMARY KEY,
  owner           VARCHAR(255) NOT NULL,
  repo            VARCHAR(255) NOT NULL,
  installation_id BIGINT NOT NULL,
  created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT unique_owner_repo UNIQUE (owner, repo)
);

-- ---------------------------------------------------------------------------
-- 3. AST Storage
-- ---------------------------------------------------------------------------

-- 3a. AST Function Symbol Storage (indexed for <1ms lookups)
CREATE TABLE IF NOT EXISTS ast_nodes (
  id            VARCHAR(64)  PRIMARY KEY,
  owner         VARCHAR(255) NOT NULL,
  repo          VARCHAR(255) NOT NULL,
  commit_sha    VARCHAR(64)  NOT NULL,
  file_path     TEXT         NOT NULL,
  line_number   INT          NOT NULL,
  function_name VARCHAR(255),
  snippet       TEXT         NOT NULL,
  created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_ast_nodes_lookup ON ast_nodes (owner, repo, file_path, line_number);

-- 3b. Repository AST Indexing Jobs Tracker
CREATE TABLE IF NOT EXISTS ast_indexes (
  id                   VARCHAR(64)  PRIMARY KEY,
  owner                VARCHAR(255) NOT NULL,
  repo                 VARCHAR(255) NOT NULL,
  commit_sha           VARCHAR(64)  NOT NULL,
  branch               VARCHAR(128) DEFAULT 'main',
  status               VARCHAR(32)  DEFAULT 'PENDING', -- PENDING | PROCESSING | INDEXED | FAILED
  parsed_files_count   INT          DEFAULT 0,
  total_functions_count INT         DEFAULT 0,
  error_message        TEXT,
  indexed_at           TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------------------------
-- 4. Ingested Incidents & AI Analysis Log
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS incidents (
  id                  VARCHAR(64) PRIMARY KEY,
  repository_id       VARCHAR(64) REFERENCES repositories(id),
  title               TEXT        NOT NULL,
  status              VARCHAR(32) DEFAULT 'CRITICAL',
  file                TEXT        NOT NULL,
  line                INT         NOT NULL,
  panic_message       TEXT        NOT NULL,
  stack_trace         TEXT        NOT NULL,
  ast_snippet         TEXT,
  root_cause          TEXT,
  suggested_fix       TEXT,
  github_issue_url    TEXT,
  github_issue_number INT,
  created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------------------------
-- 5. API Keys
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS api_keys (
  id            VARCHAR(64)  PRIMARY KEY,
  user_id       VARCHAR(64)  REFERENCES users(id),
  repository_id VARCHAR(64)  REFERENCES repositories(id),
  name          VARCHAR(255) NOT NULL,
  key_hash      VARCHAR(255) UNIQUE NOT NULL,
  key_masked    VARCHAR(64)  NOT NULL,
  revoked_at    TIMESTAMP WITH TIME ZONE,
  expires_at    TIMESTAMP WITH TIME ZONE,
  created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------------------------
-- 6. Webhook Delivery Idempotency & Audit Logs
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS webhook_logs (
  id            VARCHAR(64)  PRIMARY KEY,
  delivery_id   VARCHAR(255) UNIQUE,
  event_type    VARCHAR(128) NOT NULL,
  status        VARCHAR(32)  DEFAULT 'SUCCESS',
  status_code   INT          DEFAULT 200,
  request_body  TEXT,
  response_body TEXT,
  created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_created_at ON webhook_logs (created_at);

-- ---------------------------------------------------------------------------
-- 7. Instance Configuration (KV store — GitHub App & OAuth credentials)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS instance_config (
  key        VARCHAR(128) PRIMARY KEY,
  value      TEXT         NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------------------------
-- 8. GitHub App Installations (org → installation mapping)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS github_installations (
  id              VARCHAR(64) PRIMARY KEY,
  installation_id BIGINT      UNIQUE NOT NULL,
  org_login       VARCHAR(255) NOT NULL,
  org_id          BIGINT,
  account_type    VARCHAR(32) DEFAULT 'Organization',
  status          VARCHAR(32) DEFAULT 'active',
  created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Repos accessible via a GitHub App installation
CREATE TABLE IF NOT EXISTS installation_repos (
  id              VARCHAR(64) PRIMARY KEY,
  installation_id BIGINT NOT NULL REFERENCES github_installations(installation_id),
  owner           VARCHAR(255) NOT NULL,
  repo            VARCHAR(255) NOT NULL,
  created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT unique_install_repo UNIQUE (installation_id, owner, repo)
);
