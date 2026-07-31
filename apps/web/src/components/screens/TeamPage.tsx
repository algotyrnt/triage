/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState } from 'react';
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
  members: TeamMember[];
  onNavigate: (screen: ScreenId) => void;
}

export const TeamPage: React.FC<TeamPageProps> = ({ members, onNavigate }) => {
  const [memberList, setMemberList] = useState<TeamMember[]>(members);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [newGithubUser, setNewGithubUser] = useState('');
  const [newRole, setNewRole] = useState<'Admin' | 'Member' | 'Read-Only'>('Member');

  // Security Toggles
  const [orgSync, setOrgSync] = useState(true);
  const [require2FA, setRequire2FA] = useState(true);
  const [restrictedAst, setRestrictedAst] = useState(false);

  const handleAddMember = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newGithubUser.trim()) return;

    const newMember: TeamMember = {
      id: `usr-${Date.now()}`,
      name: newGithubUser,
      githubUsername: newGithubUser.replace('@', ''),
      role: newRole,
      scopes: ['repo:read', 'incidents:read'],
      lastActive: 'Invited just now',
      mfaEnabled: true,
    };

    setMemberList((prev) => [...prev, newMember]);
    setNewGithubUser('');
    setShowInviteModal(false);
  };

  const handleRemoveMember = (id: string) => {
    setMemberList((prev) => prev.filter((m) => m.id !== id));
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-bold text-slate-900 tracking-tight font-sans">
            Team Members & Access Permissions
          </h1>
          <p className="text-xs text-slate-600 font-sans mt-0.5">
            Manage organization team access, GitHub profile synchronization, and granular RBAC scopes.
          </p>
        </div>

        <button
          onClick={() => setShowInviteModal(true)}
          className="bg-black hover:bg-slate-800 text-white font-mono text-xs font-semibold py-2 px-3.5 rounded-sm transition-colors flex items-center gap-1.5 cursor-pointer self-start sm:self-auto"
        >
          <UserPlus className="w-3.5 h-3.5" />
          <span>Invite Team Member</span>
        </button>
      </div>

      {/* Security Toggle Card with Crisp Black Switches */}
      <div className="bg-white border border-slate-200 rounded-sm p-4 space-y-4">
        <div className="flex items-center gap-2 font-mono text-xs font-bold text-slate-900 border-b border-slate-100 pb-2.5">
          <Shield className="w-4 h-4 text-slate-800" />
          <span>Organization Security & Access Controls</span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 font-mono text-xs">
          {/* Switch 1: GitHub Org Sync */}
          <div className="bg-slate-50 border border-slate-200 p-3 rounded-sm flex items-start justify-between gap-3">
            <div className="space-y-1">
              <div className="font-bold text-slate-900">Enforce GitHub Org Sync</div>
              <div className="text-[11px] text-slate-600 leading-tight">
                Automatically revoke Triage access when removed from GitHub organization.
              </div>
            </div>
            <button
              onClick={() => setOrgSync(!orgSync)}
              className={`w-10 h-5 rounded-full p-0.5 transition-colors shrink-0 cursor-pointer ${
                orgSync ? 'bg-black' : 'bg-slate-300'
              }`}
            >
              <div
                className={`w-4 h-4 rounded-full bg-white transition-transform ${
                  orgSync ? 'translate-x-5' : 'translate-x-0'
                }`}
              ></div>
            </button>
          </div>

          {/* Switch 2: Require 2FA */}
          <div className="bg-slate-50 border border-slate-200 p-3 rounded-sm flex items-start justify-between gap-3">
            <div className="space-y-1">
              <div className="font-bold text-slate-900">Require 2FA for Triage</div>
              <div className="text-[11px] text-slate-600 leading-tight">
                Mandate hardware key or TOTP 2FA for incident triage and AST export.
              </div>
            </div>
            <button
              onClick={() => setRequire2FA(!require2FA)}
              className={`w-10 h-5 rounded-full p-0.5 transition-colors shrink-0 cursor-pointer ${
                require2FA ? 'bg-black' : 'bg-slate-300'
              }`}
            >
              <div
                className={`w-4 h-4 rounded-full bg-white transition-transform ${
                  require2FA ? 'translate-x-5' : 'translate-x-0'
                }`}
              ></div>
            </button>
          </div>

          {/* Switch 3: Restricted AST Scope */}
          <div className="bg-slate-50 border border-slate-200 p-3 rounded-sm flex items-start justify-between gap-3">
            <div className="space-y-1">
              <div className="font-bold text-slate-900">Restricted AST Scope</div>
              <div className="text-[11px] text-slate-600 leading-tight">
                Mask internal bytecode offsets and restrict AST export to exported packages.
              </div>
            </div>
            <button
              onClick={() => setRestrictedAst(!restrictedAst)}
              className={`w-10 h-5 rounded-full p-0.5 transition-colors shrink-0 cursor-pointer ${
                restrictedAst ? 'bg-black' : 'bg-slate-300'
              }`}
            >
              <div
                className={`w-4 h-4 rounded-full bg-white transition-transform ${
                  restrictedAst ? 'translate-x-5' : 'translate-x-0'
                }`}
              ></div>
            </button>
          </div>
        </div>
      </div>

      {/* Member Management Table */}
      <div className="bg-white border border-slate-200 rounded-sm overflow-hidden">
        <div className="bg-slate-100 border-b border-slate-200 p-3 font-mono text-xs font-bold text-slate-900 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Users className="w-4 h-4 text-slate-800" />
            <span>Active Team Members ({memberList.length})</span>
          </div>
          <span className="text-slate-500 font-normal">Org: algotyrnt</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left font-mono text-xs">
            <thead className="bg-slate-50 border-b border-slate-200 text-slate-500 text-[11px] uppercase tracking-wider">
              <tr>
                <th className="py-2.5 px-4 font-semibold">User / GitHub Profile</th>
                <th className="py-2.5 px-4 font-semibold">Role</th>
                <th className="py-2.5 px-4 font-semibold">Access Scopes</th>
                <th className="py-2.5 px-4 font-semibold">2FA</th>
                <th className="py-2.5 px-4 font-semibold">Last Active</th>
                <th className="py-2.5 px-4 font-semibold text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 text-slate-800">
              {memberList.map((member) => (
                <tr key={member.id} className="hover:bg-slate-50 transition-colors">
                  <td className="py-3 px-4">
                    <div className="flex items-center gap-2.5">
                      <div className="w-7 h-7 bg-slate-900 text-white font-bold text-xs rounded-sm flex items-center justify-center font-mono">
                        {member.githubUsername.substring(0, 2).toUpperCase()}
                      </div>
                      <div>
                        <div className="font-bold text-slate-900">{member.name}</div>
                        <div className="text-[11px] text-slate-500 flex items-center gap-1">
                          <Github className="w-3 h-3" />
                          <span>@{member.githubUsername}</span>
                        </div>
                      </div>
                    </div>
                  </td>

                  <td className="py-3 px-4">
                    <span
                      className={`text-[11px] font-bold px-2 py-0.5 rounded-sm border ${
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
                          className="bg-slate-100 text-slate-600 border border-slate-200 text-[10px] px-1.5 py-0.2 rounded-sm"
                        >
                          {s}
                        </span>
                      ))}
                    </div>
                  </td>

                  <td className="py-3 px-4">
                    <span className="text-emerald-700 font-bold text-[11px] flex items-center gap-1">
                      <Check className="w-3 h-3 text-emerald-600" />
                      <span>2FA Enforced</span>
                    </span>
                  </td>

                  <td className="py-3 px-4 text-slate-500 text-[11px]">{member.lastActive}</td>

                  <td className="py-3 px-4 text-right">
                    {member.role !== 'Owner' ? (
                      <button
                        onClick={() => handleRemoveMember(member.id)}
                        className="text-red-600 hover:text-red-800 text-xs font-mono underline"
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
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4">
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
                  type="text"
                  value={newGithubUser}
                  onChange={(e) => setNewGithubUser(e.target.value)}
                  placeholder="e.g. devonvance"
                  required
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-black"
                />
              </div>

              <div className="space-y-1">
                <label className="font-bold text-slate-700 block">Select Role:</label>
                <select
                  value={newRole}
                  onChange={(e) => setNewRole(e.target.value as any)}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-sm font-mono focus:bg-white focus:outline-none focus:border-black"
                >
                  <option value="Admin">Admin (Manage AST & Webhooks)</option>
                  <option value="Member">Member (Read & Triage Incidents)</option>
                  <option value="Read-Only">Read-Only (View Only)</option>
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
                  className="bg-black hover:bg-slate-800 text-white font-bold px-4 py-1.5 rounded-sm border border-black cursor-pointer"
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
