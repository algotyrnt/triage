/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect, useRef } from 'react';
import { TeamMember, ScreenId } from '@/types';
import { GithubIcon as Github } from '@/components/GithubIcon';
import { engineClient } from '@/services/engineClient';
import { Users, Shield, Lock, UserPlus, X, Mail, CheckCircle2, AlertCircle } from 'lucide-react';

interface TeamPageProps {
  teamMembers?: TeamMember[];
  currentUser?: { username: string; avatarUrl?: string; role?: string } | null;
  onNavigate: (screen: ScreenId) => void;
}

interface PendingInvite {
  id: string;
  githubUsername: string;
  role: string;
  sentAt: string;
}

export const TeamPage: React.FC<TeamPageProps> = ({ currentUser, onNavigate }) => {
  const [members, setMembers] = useState<any[]>([]);
  const [pendingInvites, setPendingInvites] = useState<PendingInvite[]>([]);
  const [loading, setLoading] = useState(true);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [inviteUsername, setInviteUsername] = useState('');
  const [inviteRole, setInviteRole] = useState<'Admin' | 'Developer' | 'Viewer'>('Developer');
  const [searchQuery, setSearchQuery] = useState('');
  const [errorMsg, setErrorMsg] = useState('');
  const [successMsg, setSuccessMsg] = useState('');

  const inviteTriggerRef = useRef<HTMLButtonElement | null>(null);
  const usernameInputRef = useRef<HTMLInputElement | null>(null);
  const modalRef = useRef<HTMLDivElement | null>(null);

  const isOwnerOrAdmin =
    currentUser?.role?.toLowerCase() === 'owner' || currentUser?.role?.toLowerCase() === 'admin';

  const loadTeamData = async () => {
    setLoading(true);
    try {
      const [membersData, invitesData] = await Promise.all([
        engineClient.getTeamMembers(),
        engineClient.getTeamInvites(),
      ]);

      if (membersData) {
        setMembers(membersData);
      }
      if (invitesData) {
        setPendingInvites(
          invitesData.map((inv: any) => ({
            id: inv.id,
            githubUsername: inv.github_username,
            role: inv.role,
            sentAt: new Date(inv.created_at || Date.now()).toLocaleDateString(),
          })),
        );
      }
    } catch (e) {
      console.error('Failed to load team data', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTeamData();
  }, []);

  useEffect(() => {
    if (showInviteModal) {
      usernameInputRef.current?.focus();
      const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
          setShowInviteModal(false);
          return;
        }
        if (e.key === 'Tab' && modalRef.current) {
          const focusables = modalRef.current.querySelectorAll<HTMLElement>(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
          );
          if (focusables.length === 0) return;
          const first = focusables[0];
          const last = focusables[focusables.length - 1];
          if (e.shiftKey && document.activeElement === first) {
            e.preventDefault();
            last.focus();
          } else if (!e.shiftKey && document.activeElement === last) {
            e.preventDefault();
            first.focus();
          }
        }
      };
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    } else {
      inviteTriggerRef.current?.focus();
    }
  }, [showInviteModal]);

  const filteredMembers = members.filter(
    (m) =>
      (m.username || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
      (m.email || '').toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteUsername) return;
    setErrorMsg('');
    setSuccessMsg('');

    const res = await engineClient.createInvite(inviteUsername.trim(), inviteRole);
    if (res.success) {
      setSuccessMsg(`Invitation created for @${inviteUsername.trim()}`);
      setInviteUsername('');
      setShowInviteModal(false);
      loadTeamData();
    } else {
      setErrorMsg(res.error || 'Failed to create invitation');
    }
  };

  const handleCancelInvite = async (id: string) => {
    setErrorMsg('');
    const res = await engineClient.cancelInvite(id);
    if (res.success) {
      setPendingInvites((prev) => prev.filter((inv) => inv.id !== id));
    } else {
      setErrorMsg(res.error || 'Failed to revoke invitation');
    }
  };

  const handleRemoveMember = async (id: string, username: string) => {
    if (!window.confirm(`Are you sure you want to revoke access for @${username}?`)) return;
    setErrorMsg('');
    const res = await engineClient.removeMember(id);
    if (res.success) {
      setMembers((prev) => prev.filter((m) => m.id !== id));
      setSuccessMsg(`Access revoked for @${username}`);
    } else {
      setErrorMsg(res.error || 'Failed to remove member');
    }
  };

  const handleRoleChange = async (id: string, newRole: string) => {
    setErrorMsg('');
    const res = await engineClient.updateMemberRole(id, newRole);
    if (res.success) {
      setMembers((prev) => prev.map((m) => (m.id === id ? { ...m, role: newRole } : m)));
      setSuccessMsg('Member role updated');
    } else {
      setErrorMsg(res.error || 'Failed to update role');
      loadTeamData();
    }
  };

  return (
    <div className="relative">
      <div
        aria-hidden={showInviteModal}
        className={`max-w-7xl mx-auto px-4 py-6 space-y-6 ${showInviteModal ? 'pointer-events-none select-none' : ''}`}
      >
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
          <div>
            <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
              Team Access Control & RBAC Permissions
            </h1>
            <p className="text-xs text-slate-600 font-sans mt-0.5">
              Manage organization team members, AST symbolication scopes, and 2FA enforcement
              policy.
            </p>
          </div>

          {isOwnerOrAdmin && (
            <button
              ref={inviteTriggerRef}
              onClick={() => setShowInviteModal(true)}
              className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-1.5 px-3 rounded-sm transition-colors flex items-center gap-1.5 cursor-pointer self-start sm:self-auto"
            >
              <UserPlus className="w-3.5 h-3.5" />
              <span>Invite GitHub User</span>
            </button>
          )}
        </div>

        {/* Feedback alerts */}
        {errorMsg && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-3 py-2 rounded-sm text-xs font-mono flex items-center justify-between">
            <div className="flex items-center gap-2">
              <AlertCircle className="w-4 h-4 text-red-600" />
              <span>{errorMsg}</span>
            </div>
            <button onClick={() => setErrorMsg('')} className="text-red-400 hover:text-red-800">
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 px-3 py-2 rounded-sm text-xs font-mono flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600" />
              <span>{successMsg}</span>
            </div>
            <button
              onClick={() => setSuccessMsg('')}
              className="text-emerald-400 hover:text-emerald-800"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {/* Security Policies Overview Bar */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-white border border-slate-200 p-3 rounded-sm flex items-center gap-3">
            <div className="p-2 bg-slate-100 rounded-sm">
              <Shield className="w-4 h-4 text-slate-800" />
            </div>
            <div className="font-mono text-xs">
              <div className="font-bold text-slate-900">Enforced RBAC Policy</div>
              <div className="text-[11px] text-slate-500">
                Zero-trust cryptographic JWT sessions
              </div>
            </div>
          </div>

          <div className="bg-white border border-slate-200 p-3 rounded-sm flex items-center gap-3">
            <div className="p-2 bg-slate-100 rounded-sm">
              <Lock className="w-4 h-4 text-slate-800" />
            </div>
            <div className="font-mono text-xs">
              <div className="font-bold text-slate-900">Role Scoping</div>
              <div className="text-[11px] text-slate-500">
                Owner, Admin, Developer, &amp; Viewer tiers
              </div>
            </div>
          </div>

          <div className="bg-white border border-slate-200 p-3 rounded-sm flex items-center gap-3">
            <div className="p-2 bg-slate-100 rounded-sm">
              <Users className="w-4 h-4 text-slate-800" />
            </div>
            <div className="font-mono text-xs">
              <div className="font-bold text-slate-900">{members.length} Active Members</div>
              <div className="text-[11px] text-slate-500">
                Your Role:{' '}
                <span className="font-bold text-slate-900">{currentUser?.role || 'Developer'}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Pending Invitations Section */}
        {pendingInvites.length > 0 && (
          <div className="bg-amber-50 border border-amber-200 rounded-sm p-4 font-mono text-xs space-y-3">
            <div className="font-bold text-amber-900 flex items-center gap-2">
              <Mail className="w-4 h-4" />
              <span>Pending Invitations ({pendingInvites.length})</span>
            </div>
            <div className="space-y-2">
              {pendingInvites.map((inv) => (
                <div
                  key={inv.id}
                  className="bg-white border border-amber-200 p-2.5 rounded-sm flex items-center justify-between"
                >
                  <div className="flex items-center gap-2">
                    <span className="font-bold text-slate-900">@{inv.githubUsername}</span>
                    <span className="bg-amber-100 text-amber-800 px-1.5 py-0.5 rounded text-[10px]">
                      {inv.role}
                    </span>
                    <span className="text-slate-500 text-[11px]">
                      • Invited {inv.sentAt} (Awaiting first GitHub login)
                    </span>
                  </div>
                  {isOwnerOrAdmin && (
                    <button
                      onClick={() => handleCancelInvite(inv.id)}
                      className="text-xs text-red-600 hover:underline cursor-pointer"
                    >
                      Revoke Invite
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Team Member List */}
        <div className="bg-white border border-slate-200 rounded-sm overflow-hidden font-mono text-xs">
          <div className="bg-slate-100 border-b border-slate-200 p-3 font-bold text-slate-900 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Users className="w-4 h-4 text-slate-700" />
              <span>Organization Member Roster ({filteredMembers.length})</span>
            </div>

            <div className="w-48">
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                aria-label="Filter members"
                placeholder="Filter members..."
                className="w-full px-2 py-1 bg-white border border-slate-200 rounded-sm text-xs font-mono focus:outline-none focus:border-black"
              />
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead className="bg-slate-50 border-b border-slate-200 text-slate-500 text-[11px] uppercase tracking-wider">
                <tr>
                  <th className="py-2.5 px-4 font-semibold">User</th>
                  <th className="py-2.5 px-4 font-semibold">Role</th>
                  <th className="py-2.5 px-4 font-semibold">Email</th>
                  <th className="py-2.5 px-4 font-semibold">Joined</th>
                  <th className="py-2.5 px-4 font-semibold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200">
                {filteredMembers.map((m) => (
                  <tr key={m.id} className="hover:bg-slate-50">
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2.5">
                        {m.avatar_url ? (
                          <img
                            src={m.avatar_url}
                            alt=""
                            className="w-7 h-7 rounded-full border border-slate-200"
                          />
                        ) : (
                          <div className="w-7 h-7 rounded-full bg-slate-200 flex items-center justify-center font-bold text-slate-700 text-xs">
                            {(m.username || 'U').charAt(0).toUpperCase()}
                          </div>
                        )}
                        <div>
                          <div className="font-bold text-slate-900 flex items-center gap-1">
                            <span>{m.username}</span>
                            <a
                              href={`https://github.com/${m.username}`}
                              target="_blank"
                              rel="noreferrer"
                              className="text-slate-400 hover:text-black"
                            >
                              <Github className="w-3 h-3" />
                            </a>
                          </div>
                          <div className="text-[11px] text-slate-500">ID: {m.id}</div>
                        </div>
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      {isOwnerOrAdmin && m.role !== 'Owner' ? (
                        <select
                          value={m.role}
                          onChange={(e) => handleRoleChange(m.id, e.target.value)}
                          className="bg-slate-50 border border-slate-300 rounded px-1.5 py-0.5 text-xs font-mono font-bold"
                        >
                          <option value="Admin">Admin</option>
                          <option value="Developer">Developer</option>
                          <option value="Viewer">Viewer</option>
                        </select>
                      ) : (
                        <span
                          className={`px-2 py-0.5 rounded-sm font-bold text-[10px] border ${
                            m.role === 'Owner'
                              ? 'bg-purple-50 text-purple-700 border-purple-200'
                              : m.role === 'Admin'
                                ? 'bg-indigo-50 text-indigo-700 border-indigo-200'
                                : m.role === 'Developer'
                                  ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                                  : 'bg-slate-100 text-slate-700 border-slate-200'
                          }`}
                        >
                          {m.role}
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-4 text-slate-600">{m.email || '—'}</td>
                    <td className="py-3 px-4 text-slate-500 text-[11px]">
                      {m.created_at ? new Date(m.created_at).toLocaleDateString() : '—'}
                    </td>
                    <td className="py-3 px-4 text-right">
                      {isOwnerOrAdmin && m.role !== 'Owner' && m.id !== currentUser?.username && (
                        <button
                          onClick={() => handleRemoveMember(m.id, m.username)}
                          className="text-red-600 hover:underline text-xs cursor-pointer"
                        >
                          Revoke Access
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Invite Modal */}
      {showInviteModal && (
        <div
          ref={modalRef}
          role="dialog"
          aria-modal="true"
          aria-label="Invite GitHub User"
          className="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4"
        >
          <div className="bg-white border border-slate-200 rounded-sm p-6 w-full max-w-md space-y-4 shadow-none font-mono">
            <div className="flex items-center justify-between border-b border-slate-200 pb-3">
              <div className="font-bold text-sm text-slate-900">Invite GitHub User</div>
              <button
                onClick={() => setShowInviteModal(false)}
                className="text-slate-400 hover:text-black cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleAddMember} className="space-y-4 text-xs">
              <div className="space-y-1">
                <label className="font-bold text-slate-700 block">GitHub Username:</label>
                <input
                  ref={usernameInputRef}
                  type="text"
                  value={inviteUsername}
                  onChange={(e) => setInviteUsername(e.target.value)}
                  placeholder="e.g. torvalds"
                  required
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-black"
                />
              </div>

              <div className="space-y-1">
                <label className="font-bold text-slate-700 block">Assigned Role:</label>
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value as any)}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-black"
                >
                  <option value="Developer">Developer (Triage, ASTs, Fix PRs)</option>
                  <option value="Admin">Admin (Project Management &amp; Invites)</option>
                  <option value="Viewer">Viewer (Read-only Dashboards)</option>
                </select>
              </div>

              <div className="pt-2 flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowInviteModal(false)}
                  className="bg-slate-100 text-slate-700 px-3 py-1.5 rounded-sm border border-slate-200 cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="bg-black hover:bg-slate-800 text-white font-bold px-4 py-1.5 rounded-sm transition-colors cursor-pointer"
                >
                  Send Invitation
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
