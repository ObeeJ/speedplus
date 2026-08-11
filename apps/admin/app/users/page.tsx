'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi } from '@speedplus/api-client';
import { UsersIcon, DriverIcon, StoreIcon, KYCIcon } from '@speedplus/ui';

type UserRow = Awaited<ReturnType<typeof adminApi.listUsers>>['users'][number];

type RoleMeta = { label: string; bg: string; text: string; Icon: typeof UsersIcon };

const ROLE_META: Record<string, RoleMeta> = {
  customer: { label: 'Customer', bg: '#E9F3D8', text: '#0A3D2C', Icon: UsersIcon },
  driver:   { label: 'Driver',   bg: '#FFF7E6', text: '#8A6A1B', Icon: DriverIcon },
  merchant: { label: 'Merchant', bg: '#EEF2FF', text: '#3730A3', Icon: StoreIcon },
  admin:    { label: 'Admin',    bg: '#F3F4F6', text: '#374151', Icon: KYCIcon },
};

function getRoleMeta(role: string): RoleMeta {
  return ROLE_META[role] ?? ROLE_META['customer']!;
}

const ROLE_FILTERS = ['', 'customer', 'driver', 'merchant'];

export default function UsersPage() {
  const [users, setUsers] = useState<UserRow[]>([]);
  const [role, setRole] = useState('');
  const [q, setQ] = useState('');
  const [error, setError] = useState('');
  const [, startTransition] = useTransition();

  useEffect(() => {
    startTransition(async () => {
      try {
        const d = await adminApi.listUsers(role || undefined, q || undefined);
        setError('');
        setUsers(d.users ?? []);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to load users');
      }
    });
  }, [role, q]);

  const visible = users.filter((u) => {
    if (role && u.role !== role) return false;
    if (q) {
      const lq = q.toLowerCase();
      return (
        u.firstName.toLowerCase().includes(lq) ||
        u.lastName.toLowerCase().includes(lq) ||
        u.phone.includes(lq) ||
        (u.email ?? '').toLowerCase().includes(lq)
      );
    }
    return true;
  });

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="w-9 h-9 rounded-[11px] flex items-center justify-center" style={{ background: '#E9F3D8' }}>
          <UsersIcon size={18} color="#0A3D2C" accent="#7BA05B" />
        </div>
        <div>
          <h1 className="font-display font-semibold text-[22px] tracking-tight text-ink">Users</h1>
          <p className="text-[12px] text-mid">All registered accounts</p>
        </div>
      </div>

      {/* Search + role filter */}
      <div className="flex gap-3 flex-wrap items-center">
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search name, phone, email…"
          className="h-9 rounded-[11px] border border-line bg-white px-3.5 text-[13px] text-ink placeholder:text-mid focus:outline-none focus:border-[#0A3D2C]/40 w-64" aria-label="Search name, phone, email"/>
        <div className="flex gap-2">
          {ROLE_FILTERS.map((r) => (
            <button
              key={r}
              onClick={() => setRole(r)}
              className={`text-[11.5px] font-semibold rounded-full px-3.5 py-1.5 transition-colors border ${
                role === r
                  ? 'bg-[#0A3D2C] text-[#F7F5EF] border-[#0A3D2C]'
                  : 'bg-white text-mid border-line hover:border-[#0A3D2C]/30 hover:text-ink'
              }`}
            >
              {r ? r.charAt(0).toUpperCase() + r.slice(1) : 'All'}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div className="bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-4 py-3 text-sm text-[#DC2626]">
          {error}
        </div>
      )}

      {/* Table */}
      {visible.length === 0 && !error ? (
        <div className="flex flex-col items-center justify-center py-20 gap-4">
          <div className="w-14 h-14 rounded-2xl flex items-center justify-center" style={{ background: '#E9F3D8' }}>
            <UsersIcon size={26} color="#0A3D2C" accent="#7BA05B" />
          </div>
          <p className="text-[13px] text-mid">No users found.</p>
        </div>
      ) : (
        <div className="bg-white border border-line rounded-2xl overflow-hidden">
          <table className="w-full text-[13px]">
            <thead>
              <tr className="border-b border-line">
                <th className="text-left px-5 py-3 text-[10.5px] font-semibold text-mid tracking-[0.5px] uppercase">Name</th>
                <th className="text-left px-5 py-3 text-[10.5px] font-semibold text-mid tracking-[0.5px] uppercase">Phone</th>
                <th className="text-left px-5 py-3 text-[10.5px] font-semibold text-mid tracking-[0.5px] uppercase">Role</th>
                <th className="text-left px-5 py-3 text-[10.5px] font-semibold text-mid tracking-[0.5px] uppercase">Status</th>
                <th className="text-left px-5 py-3 text-[10.5px] font-semibold text-mid tracking-[0.5px] uppercase">Joined</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((u, i) => {
                const meta = getRoleMeta(u.role);
                const { Icon } = meta;
                return (
                  <tr
                    key={u.id}
                    className={`border-b border-line last:border-0 hover:bg-[#F7F5EF]/60 transition-colors ${
                      i % 2 === 0 ? '' : 'bg-[#FAFAF8]'
                    }`}
                  >
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-2.5">
                        <div
                          className="w-7 h-7 rounded-full flex items-center justify-center shrink-0"
                          style={{ background: meta.bg }}
                        >
                          <Icon size={13} color={meta.text} accent={meta.text} />
                        </div>
                        <div>
                          <div className="font-semibold text-ink">{u.firstName} {u.lastName}</div>
                          {u.email && <div className="text-[11px] text-mid">{u.email}</div>}
                        </div>
                      </div>
                    </td>
                    <td className="px-5 py-3.5 font-mono text-[12.5px] text-mid">{u.phone}</td>
                    <td className="px-5 py-3.5">
                      <span
                        className="text-[10.5px] font-bold rounded-full px-2.5 py-0.5"
                        style={{ background: meta.bg, color: meta.text }}
                      >
                        {meta.label}
                      </span>
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-1.5">
                        <div
                          className="w-1.5 h-1.5 rounded-full"
                          style={{ background: u.isActive ? '#C6F24E' : '#E4E0D6' }}
                        />
                        <span className="text-[12px] text-mid">
                          {u.isActive ? 'Active' : 'Inactive'}
                          {!u.isVerified && ' · Unverified'}
                        </span>
                      </div>
                    </td>
                    <td className="px-5 py-3.5 text-[12px] text-mid">
                      {new Date(u.createdAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', year: '2-digit' })}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
