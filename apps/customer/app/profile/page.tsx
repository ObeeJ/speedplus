'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { FEATURES } from '@/lib/features';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Skeleton } from '@fourdat/ui';
import { usersApi, authApi, type CreateAddressPayload } from '@fourdat/api-client';
import { useAuthStore } from '@/lib/store/auth.store';

const BLANK: CreateAddressPayload = { label: '', street: '', city: '', state: '', lat: 0, lng: 0 };

export default function ProfilePage() {
  const router = useRouter();
  const qc = useQueryClient();
  const clearAuth = useAuthStore((s) => s.clearAuth);

  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState<CreateAddressPayload>(BLANK);
  const [formError, setFormError] = useState('');
  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState({ firstName: '', lastName: '', email: '' });
  const [editError, setEditError] = useState('');

  const [settingPin, setSettingPin] = useState(false);
  const [pin, setPinInput] = useState('');
  const [pinError, setPinError] = useState('');
  const [pinDone, setPinDone] = useState(false);

  const OTP_PURPOSE = 'phone_verification';
  const [otpStage, setOtpStage] = useState<'idle' | 'sent' | 'verified'>('idle');
  const [otpCode, setOtpCode] = useState('');
  const [otpError, setOtpError] = useState('');

  const { data: user } = useQuery({
    queryKey: ['me'],
    queryFn: () => usersApi.me(),
    staleTime: 60_000,
  });

  const updateMe = useMutation({
    mutationFn: (payload: Parameters<typeof usersApi.updateMe>[0]) => usersApi.updateMe(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['me'] });
      setEditing(false);
      setEditError('');
    },
    onError: (e: Error) => setEditError(e.message),
  });

  const { data: addresses = [], isLoading } = useQuery({
    queryKey: ['addresses'],
    queryFn: () => usersApi.listAddresses(),
    staleTime: 30_000,
  });

  const addAddress = useMutation({
    mutationFn: (payload: CreateAddressPayload) => usersApi.createAddress(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['addresses'] });
      setAdding(false);
      setForm(BLANK);
      setFormError('');
    },
    onError: (e: Error) => setFormError(e.message),
  });

  const setPinMutation = useMutation({
    mutationFn: (p: string) => authApi.setPin(p),
    onSuccess: () => {
      setPinDone(true);
      setSettingPin(false);
      setPinInput('');
      setPinError('');
    },
    onError: (e: Error) => setPinError(e.message),
  });

  function handleSetPin(e: React.FormEvent) {
    e.preventDefault();
    if (!/^\d{4}$/.test(pin)) {
      setPinError('PIN must be exactly 4 digits.');
      return;
    }
    setPinError('');
    setPinMutation.mutate(pin);
  }

  const requestOtpMutation = useMutation({
    mutationFn: () => authApi.requestOtp(user?.phone ?? '', OTP_PURPOSE),
    onSuccess: () => {
      setOtpStage('sent');
      setOtpError('');
    },
    onError: (e: Error) => setOtpError(e.message),
  });

  const verifyOtpMutation = useMutation({
    mutationFn: () => authApi.verifyOtpCode(user?.phone ?? '', otpCode, OTP_PURPOSE),
    onSuccess: (res) => {
      if (res.verified) {
        setOtpStage('verified');
        setOtpError('');
      } else {
        setOtpError('Incorrect code. Try again.');
      }
    },
    onError: (e: Error) => setOtpError(e.message),
  });

  async function handleLogout() {
    await authApi.logout().catch(() => {});
    clearAuth();
    router.replace('/login');
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.street.trim() || !form.city.trim() || !form.state.trim()) {
      setFormError('Street, city, and state are required.');
      return;
    }
    if (!form.lat || !form.lng) {
      setFormError('Latitude and longitude are required.');
      return;
    }
    setFormError('');
    addAddress.mutate({ ...form, label: form.label?.trim() || undefined });
  }

  function startEdit() {
    setEditForm({ firstName: user?.firstName ?? '', lastName: user?.lastName ?? '', email: user?.email ?? '' });
    setEditing(true);
  }

  function handleEditSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!editForm.firstName.trim() || !editForm.lastName.trim()) {
      setEditError('First and last name are required.');
      return;
    }
    setEditError('');
    updateMe.mutate({
      firstName: editForm.firstName.trim(),
      lastName: editForm.lastName.trim(),
      ...(editForm.email.trim() ? { email: editForm.email.trim() } : {}),
    });
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <div className="px-5 pt-12 pb-4 flex items-center justify-between">
        <button onClick={() => router.back()} className="text-mid hover:text-ink transition-colors" aria-label="Back">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M19 12H5M12 5l-7 7 7 7" /></svg>
        </button>
        <h1 className="font-display font-semibold text-[17px] text-ink">Profile</h1>
        <div className="w-5" />
      </div>

      <div className="flex-1 px-5 py-4 flex flex-col gap-6 min-[700px]:max-w-[640px] min-[700px]:mx-auto min-[700px]:w-full">

        {/* User details */}
        <section className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-3">
          {!editing ? (
            <>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-[15px] font-semibold text-ink">{user ? `${user.firstName} ${user.lastName}` : '—'}</p>
                  <p className="text-[12px] text-mid">{user?.phone}</p>
                  {user?.email && <p className="text-[12px] text-mid">{user.email}</p>}
                </div>
                <button onClick={startEdit} className="text-[12px] font-semibold text-emerald hover:underline">Edit</button>
              </div>
            </>
          ) : (
            <form onSubmit={handleEditSubmit} className="flex flex-col gap-3">
              <div className="grid grid-cols-2 gap-2">
                <input
                  placeholder="First name *"
                  required
                  value={editForm.firstName}
                  onChange={(e) => setEditForm((f) => ({ ...f, firstName: e.target.value }))}
                  className="border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="First name *"/>
                <input
                  placeholder="Last name *"
                  required
                  value={editForm.lastName}
                  onChange={(e) => setEditForm((f) => ({ ...f, lastName: e.target.value }))}
                  className="border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Last name *"/>
              </div>
              <input
                placeholder="Email (optional)"
                type="email"
                value={editForm.email}
                onChange={(e) => setEditForm((f) => ({ ...f, email: e.target.value }))}
                className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Email (optional)"/>
              {editError && <p className="text-xs text-red-600" role="alert">{editError}</p>}
              <div className="flex gap-2">
                <Button type="submit" variant="primary" size="sm" isLoading={updateMe.isPending} className="flex-1">Save</Button>
                <Button type="button" variant="ghost" size="sm" onClick={() => { setEditing(false); setEditError(''); }} className="flex-1">Cancel</Button>
              </div>
            </form>
          )}
        </section>

        {/* Cylinder registry link */}
        <section>
          <Link href="/cylinders" className="flex items-center justify-between bg-white border border-line rounded-2xl px-4 py-3 hover:border-emerald/40 transition-colors">
            <span className="text-[13px] font-semibold text-ink">My cylinders</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M9 18l6-6-6-6" /></svg>
          </Link>
          <Link href="/subscriptions" className="flex items-center justify-between bg-white border border-line rounded-2xl px-4 py-3 mt-2 hover:border-emerald/40 transition-colors">
            <span className="text-[13px] font-semibold text-ink">Auto-refill subscriptions</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M9 18l6-6-6-6" /></svg>
          </Link>
          {FEATURES.loyalty && (
          <Link href="/loyalty" className="flex items-center justify-between bg-white border border-line rounded-2xl px-4 py-3 mt-2 hover:border-emerald/40 transition-colors">
            <span className="text-[13px] font-semibold text-ink">Loyalty points</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M9 18l6-6-6-6" /></svg>
          </Link>
          )}
          {FEATURES.giftCards && (
          <Link href="/gift-cards" className="flex items-center justify-between bg-white border border-line rounded-2xl px-4 py-3 mt-2 hover:border-emerald/40 transition-colors">
            <span className="text-[13px] font-semibold text-ink">Gift cards</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M9 18l6-6-6-6" /></svg>
          </Link>
          )}
          <Link href="/kyc" className="flex items-center justify-between bg-white border border-line rounded-2xl px-4 py-3 mt-2 hover:border-emerald/40 transition-colors">
            <span className="text-[13px] font-semibold text-ink">Identity verification</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M9 18l6-6-6-6" /></svg>
          </Link>
        </section>

        {/* Addresses */}
        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <span className="text-[13px] font-semibold text-mid">Saved addresses</span>
            {!adding && (
              <button onClick={() => setAdding(true)} className="text-[12px] font-semibold text-emerald hover:underline">
                + Add new
              </button>
            )}
          </div>

          {isLoading && (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-[52px] rounded-2xl" />
              <Skeleton className="h-[52px] rounded-2xl" />
            </div>
          )}

          {!isLoading && addresses.length === 0 && !adding && (
            <p className="text-[13px] text-mid">No saved addresses yet. Add one to start ordering.</p>
          )}

          {addresses.map((a) => (
            <div key={a.id} className="bg-white border border-line rounded-2xl px-4 py-3">
              <p className="text-[13px] font-semibold text-ink">{a.label || a.street}</p>
              <p className="text-[11px] text-mid">{a.street}, {a.city}</p>
            </div>
          ))}

          {adding && (
            <form onSubmit={handleSubmit} className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-3">
              <input
                placeholder="Label (e.g. Home)"
                value={form.label ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, label: e.target.value }))}
                className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Label (e.g. Home)"/>
              <input
                placeholder="Street *"
                required
                value={form.street}
                onChange={(e) => setForm((f) => ({ ...f, street: e.target.value }))}
                className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Street *"/>
              <input
                placeholder="City *"
                required
                value={form.city}
                onChange={(e) => setForm((f) => ({ ...f, city: e.target.value }))}
                className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="City *"/>
              <input
                placeholder="State *"
                required
                value={form.state}
                onChange={(e) => setForm((f) => ({ ...f, state: e.target.value }))}
                className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="State *"/>
              <div className="grid grid-cols-2 gap-2">
                <input
                  placeholder="Latitude"
                  type="number"
                  step="any"
                  value={form.lat || ''}
                  onChange={(e) => setForm((f) => ({ ...f, lat: parseFloat(e.target.value) || 0 }))}
                  className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Latitude"/>
                <input
                  placeholder="Longitude"
                  type="number"
                  step="any"
                  value={form.lng || ''}
                  onChange={(e) => setForm((f) => ({ ...f, lng: parseFloat(e.target.value) || 0 }))}
                  className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Longitude"/>
              </div>
              {formError && <p className="text-xs text-red-600" role="alert">{formError}</p>}
              <div className="flex gap-2">
                <Button type="submit" variant="primary" size="sm" isLoading={addAddress.isPending} className="flex-1">
                  Save address
                </Button>
                <Button type="button" variant="ghost" size="sm" onClick={() => { setAdding(false); setForm(BLANK); setFormError(''); }} className="flex-1">
                  Cancel
                </Button>
              </div>
            </form>
          )}
        </section>

        {/* Security */}
        <section className="flex flex-col gap-3">
          <span className="text-[13px] font-semibold text-mid">Security</span>

          {/* Transaction PIN */}
          <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-2.5">
            <div className="flex items-center justify-between">
              <span className="text-[13px] font-semibold text-ink">Transaction PIN</span>
              {!settingPin && (
                <button onClick={() => { setSettingPin(true); setPinDone(false); }} className="text-[12px] font-semibold text-emerald hover:underline">
                  {pinDone ? 'Change' : 'Set PIN'}
                </button>
              )}
            </div>
            <p className="text-[11px] text-mid">Used to confirm wallet transfers and card payments at the door.</p>
            {pinDone && !settingPin && <p className="text-[12px] text-emerald font-semibold">✓ PIN set</p>}
            {settingPin && (
              <form onSubmit={handleSetPin} className="flex flex-col gap-2.5">
                <input
                  type="password"
                  inputMode="numeric"
                  maxLength={4}
                  placeholder="4-digit PIN"
                  value={pin}
                  onChange={(e) => setPinInput(e.target.value.replace(/\D/g, '').slice(0, 4))}
                  className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid tracking-[0.3em] focus:outline-none focus:border-emerald" aria-label="4-digit PIN"/>
                {pinError && <p className="text-xs text-red-600" role="alert">{pinError}</p>}
                <div className="flex gap-2">
                  <Button type="submit" variant="primary" size="sm" isLoading={setPinMutation.isPending} className="flex-1">Save PIN</Button>
                  <Button type="button" variant="ghost" size="sm" onClick={() => { setSettingPin(false); setPinInput(''); setPinError(''); }} className="flex-1">Cancel</Button>
                </div>
              </form>
            )}
          </div>

          {/* Phone verification */}
          <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-2.5">
            <div className="flex items-center justify-between">
              <span className="text-[13px] font-semibold text-ink">Phone verification</span>
              {otpStage === 'verified' && <span className="text-[12px] text-emerald font-semibold">✓ Verified</span>}
            </div>
            {otpStage === 'idle' && (
              <>
                <p className="text-[11px] text-mid">We&apos;ll text a code to {user?.phone ?? 'your phone'}.</p>
                {otpError && <p className="text-xs text-red-600" role="alert">{otpError}</p>}
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  isLoading={requestOtpMutation.isPending}
                  disabled={!user?.phone}
                  onClick={() => requestOtpMutation.mutate()}
                >
                  Send code
                </Button>
              </>
            )}
            {otpStage === 'sent' && (
              <div className="flex flex-col gap-2.5">
                <input
                  type="text"
                  inputMode="numeric"
                  placeholder="Enter code"
                  value={otpCode}
                  onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, ''))}
                  className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Enter code"/>
                {otpError && <p className="text-xs text-red-600" role="alert">{otpError}</p>}
                <div className="flex gap-2">
                  <Button type="button" variant="primary" size="sm" isLoading={verifyOtpMutation.isPending} className="flex-1" onClick={() => verifyOtpMutation.mutate()}>
                    Verify
                  </Button>
                  <Button type="button" variant="ghost" size="sm" className="flex-1" onClick={() => requestOtpMutation.mutate()}>
                    Resend
                  </Button>
                </div>
              </div>
            )}
          </div>
        </section>

        {/* Logout */}
        <section className="mt-auto pt-4 border-t border-line">
          <Button variant="ghost" size="md" onClick={handleLogout} className="w-full text-red-600 hover:bg-red-50">
            Log out
          </Button>
        </section>
      </div>
    </main>
  );
}
