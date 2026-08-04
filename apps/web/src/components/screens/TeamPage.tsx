/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect, useRef } from 'react';
import { TeamMember, ScreenId } from '../../types';
import { GithubIcon as Github } from '../GithubIcon';
import {
  Users,
  Shield,
  Plus,
  Check,
  Key,
  Lock,
  UserPlus,
  X,
  SlidersHorizontal,
} from 'lucide-react';

interface TeamPageProps {
  teamMembers: TeamMember[];
  onNavigate: (screen: ScreenId) => void;
}

export const TeamPage: React.FC<TeamPageProps> = ({ teamMembers, onNavigate }) => {
  const [members, setMembers] = useState<TeamMember[]>(teamMembers);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [inviteUsername, setInviteUsername] = useState('');
  const [inviteRole, setInviteRole] = useState<'Admin' | 'Member'>('Member');
  const [searchQuery, setSearchQuery] = useState('');

  const inviteTriggerRef = useRef<HTMLButtonElement | null>(null);
  const usernameInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (showInviteModal) {
      usernameInputRef.current?.focus();
      const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
          setShowInviteModal(false);
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
      m.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      m.githubUsername.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleAddMember = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteUsername) return;

    const newMember: TeamMember = {
      id: `usr-${Date.now()}`,
      name: inviteUsername,
      githubUsername: inviteUsername,
      avatarUrl: `https://github.com/${inviteUsername}.png`,
      role: inviteRole,
      scopes: inviteRole === 'Admin' ? ['repo:read', 'ast:write', 'incidents:manage'] : ['repo:read', 'incidents:read'],
      lastActive: 'Just invited',
      mfaEnabled: true,
    };

    setMembers((prev) => [...prev, newMember]);
    setInviteUsername('');
    setShowInviteModal(false);
  };

  const handleRemoveMember = (id: string) => {
    setMembers((prev) => prev.filter((m) => m.id !== id));
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
            Team Access Control & RBAC Permissions
          </h1>
          <p className="text-xs text-slate-600 font-sans mt-0.5">
            Manage organization team members, AST symbolication scopes, and 2FA enforcement policy.
          </p>
        </div>

        <button
          ref={inviteTriggerRef}
          onClick={() => setShowInviteModal(true)}
          className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-1.5 px-3 rounded-sm transition-colors flex items-center gap-1.5 cursor-pointer self-start sm:self-auto"
        >
          <UserPlus className="w-3.5 h-3.5" />
          <span>Invite GitHub User</span>
        </button>
      </div>

      {/* Security Policies Overview Bar */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white border border-slate-200 p-3 rounded-sm flex items-center gap-3">
          <div className="p-2 bg-slate-100 rounded-sm">
            <Shield className="w-4 h-4 text-slate-800" />
          </div>
          <div className="font-mono text-xs">
            <div className="font-bold text-slate-900">Enforced 2FA Policy</div>
            <div className="text-[11px] text-slate-500">Require GitHub 2FA for all members</div>
          </div>
        </div>

        <div className="bg-white border border-slate-200 p-3 rounded-sm flex items-center gap-3">
          <div className="p-2 bg-slate-100 rounded-sm">
            <Lock className="w-4 h-4 text-slate-800" />
          </div>
          <div className="font-mono text-xs">
            <div className="font-bold text-slate-900">Zero Trust Token Scoping</div>
            <div className="text-[11px] text-slate-500">Least privilege AST & incident access</div>
          </div>
        </div>

        <div className="bg-white border border-slate-200 p-3 rounded-sm flex items-center gap-3">
          <div className="p-2 bg-slate-100 rounded-sm">
            <Users className="w-4 h-4 text-slate-800" />
          </div>
          <div className="font-mono text-xs">
            <div className="font-bold text-slate-900">{members.length} Active Members</div>
            <div className="text-[11px] text-slate-500">Organization: algotyrnt</div>
          </div>
        </div>
      </div>

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
                <th className="py-2.5 px-4 font-semibold">Granted Scopes</th>
                <th className="py-2.5 px-4 font-semibold">2FA Security</th>
                <th className="py-2.5 px-4 font-semibold">Last Active</th>
                <th className="py-2.5 px-4 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 text-slate-800">
              {filteredMembers.map((member) => (
                <tr key={member.id} className="hover:bg-slate-50">
                  <td className="py-3 px-4">
                    <div className="flex items-center gap-2.5">
                      <img
                        src={member.avatarUrl || 'https://github.com/ghost.png'}
                        alt={member.name}
                        className="w-7 h-7 rounded-full border border-slate-300 object-cover"
                      />
                      <div>
                        <div className="font-bold text-slate-900">{member.name}</div>
                        <div className="text-[11px] text-slate-500 font-mono">@{member.githubUsername}</div>
                      </div>
                    </div>
                  </td>

                  <td className="py-3 px-4">
                    <span
                      className={`text-[10px] font-bold px-2 py-0.5 rounded-sm border ${
                        member.role === 'Owner'
                          ? 'bg-black text-white border-black'
                          : member.role === 'Admin'
                          ? 'bg-slate-800 text-white border-slate-800'
                          : 'bg-slate-100 text-slate-700 border-slate-200'
                      }`}
                    >
                      {member.role}
                    </span>
                  </td>

                  <td className="py-3 px-4">
                    <div className="flex flex-wrap gap-1">
                      {member.scopes.map((s) => (
                        <span
                          key={s}
                          className="bg-slate-100 text-slate-600 border border-slate-200 text-[10px] px-1.5 py-0.5 rounded-sm"
                        >
                          {s}
                        </span>
                      ))}
                    </div>
                  </td>

                  <td className="py-3 px-4">
                    {member.mfaEnabled ? (
                      <span className="text-emerald-700 font-bold text-[11px] flex items-center gap-1">
                        <Check className="w-3 h-3 text-emerald-600" />
                        <span>2FA Enforced</span>
                      </span>
                    ) : (
                      <span className="text-slate-400 font-normal text-[11px] flex items-center gap-1">
                        <X className="w-3 h-3 text-slate-400" />
                        <span>2FA Disabled</span>
                      </span>
                    )}
                  </td>

                  <td className="py-3 px-4 text-slate-500 text-[11px]">{member.lastActive}</td>

                  <td className="py-3 px-4 text-right">
                    {member.role !== 'Owner' ? (
                      <button
                        onClick={() => handleRemoveMember(member.id)}
                        className="text-red-600 hover:text-red-800 text-xs font-mono underline cursor-pointer"
                      >
                        Remove
                      </button>
                    ) : (
                      <span className="text-slate-400 text-[11px]">Primary Owner</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Invite Modal */}
      {showInviteModal && (
        <div
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
                className="text-slate-400 hover:text-black"
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
                  placeholder="e.g. devonvance-go"
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
                  <option value="Member">Member (Read-only Incidents)</option>
                  <option value="Admin">Admin (AST Write & Key Management)</option>
                </select>
              </div>

              <div className="pt-2 flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowInviteModal(false)}
                  className="bg-slate-100 text-slate-700 px-3 py-1.5 rounded-sm border border-slate-200"
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
