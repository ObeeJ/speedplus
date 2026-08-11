'use client';

import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { AuthShell } from './auth-shell';
import { Button } from './button';
import { Input } from './input';
import { PasswordInput } from './password-input';
import { OTPInput } from './otp-input';
import { StatusSteps } from './status-steps';
import { AlertCircleIcon } from '../icons';
import { spring } from '../lib/motion';

export interface ForgotPasswordFlowProps {
  /** AuthShell brand panel props — per-app copy */
  headline: React.ReactNode;
  subtext?: string;
  portalLabel?: string;
  /** Called to request OTP — should call authApi.requestOtp(phone, 'password_reset') */
  onRequestOtp: (phone: string) => Promise<void>;
  /** Called to verify OTP — should call authApi.verifyOtpCode(phone, otp, 'password_reset') */
  onVerifyOtp: (phone: string, otp: string) => Promise<void>;
  /** Called to set new password — should call authApi.resetPassword({ phone, otp, newPassword }) */
  onResetPassword: (phone: string, otp: string, newPassword: string) => Promise<void>;
  /** href for "Back to sign in" link */
  loginHref?: string;
}

const STEPS = [
  { label: 'Enter phone number' },
  { label: 'Verify code' },
  { label: 'Set new password' },
];

const stepHeadings = [
  { heading: 'Reset password',    sub: 'Enter your phone number to receive a verification code.' },
  { heading: 'Enter your code',   sub: 'We sent a 6-digit code to your phone.' },
  { heading: 'New password',      sub: 'Choose a strong password for your account.' },
];

export function ForgotPasswordFlow({
  headline,
  subtext,
  portalLabel,
  onRequestOtp,
  onVerifyOtp,
  onResetPassword,
  loginHref = '/login',
}: ForgotPasswordFlowProps) {
  const [step, setStep]         = useState(0);
  const [phone, setPhone]       = useState('');
  const [otp, setOtp]           = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm]   = useState('');
  const [error, setError]       = useState('');
  const [loading, setLoading]   = useState(false);
  const [done, setDone]         = useState(false);

  async function handleStep0(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await onRequestOtp(phone);
      setStep(1);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not send code. Try again.');
    } finally {
      setLoading(false);
    }
  }

  async function handleStep1(code: string) {
    setError('');
    setLoading(true);
    try {
      await onVerifyOtp(phone, code);
      setOtp(code);
      setStep(2);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid or expired code.');
    } finally {
      setLoading(false);
    }
  }

  async function handleStep2(e: React.FormEvent) {
    e.preventDefault();
    if (password.length < 8) { setError('Password must be at least 8 characters.'); return; }
    if (password !== confirm) { setError('Passwords do not match.'); return; }
    setError('');
    setLoading(true);
    try {
      await onResetPassword(phone, otp, password);
      setDone(true);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not reset password. Try again.');
    } finally {
      setLoading(false);
    }
  }

  const { heading, sub } = stepHeadings[step] ?? stepHeadings[0]!;

  return (
    <AuthShell
      headline={headline}
      subtext={subtext}
      portalLabel={portalLabel}
      formHeading={heading}
      formSubheading={sub}
    >
      {done ? (
        <motion.div
          className="flex flex-col gap-5"
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={spring.smooth}
        >
          <div className="bg-tile border border-emerald/20 rounded-xl px-4 py-4">
            <p className="text-sm text-ink font-semibold">Password updated</p>
            <p className="text-sm text-mid mt-1">You can now sign in with your new password.</p>
          </div>
          <a
            href={loginHref}
            className="text-sm text-center text-mid hover:text-ink transition-colors underline-offset-2 hover:underline"
          >
            Back to sign in
          </a>
        </motion.div>
      ) : (
        <div className="flex flex-col gap-6">
          {/* Progress */}
          <StatusSteps steps={STEPS} currentIndex={step} />

          {/* Error */}
          <AnimatePresence mode="wait">
            {error && (
              <motion.div
                key="error"
                className="flex items-start gap-2.5 bg-red-50 border border-red-200 rounded-xl px-3.5 py-2.5"
                role="alert"
                aria-live="polite"
                initial={{ opacity: 0, y: -6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -4 }}
                transition={{ duration: 0.18 }}
              >
                <AlertCircleIcon color="#DC2626" />
                <p className="text-xs text-red-600">{error}</p>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Step panels */}
          <AnimatePresence mode="wait">
            {step === 0 && (
              <motion.form
                key="step-0"
                onSubmit={handleStep0}
                className="flex flex-col gap-4"
                noValidate
                initial={{ opacity: 0, x: 16 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -16 }}
                transition={spring.smooth}
              >
                <Input
                  id="phone"
                  label="Phone number"
                  type="tel"
                  placeholder="08012345678"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  autoComplete="tel"
                  required
                />
                <Button type="submit" variant="primary" size="lg" isLoading={loading} className="w-full">
                  Send code
                </Button>
                <a
                  href={loginHref}
                  className="text-sm text-center text-mid hover:text-ink transition-colors underline-offset-2 hover:underline"
                >
                  Back to sign in
                </a>
              </motion.form>
            )}

            {step === 1 && (
              <motion.div
                key="step-1"
                className="flex flex-col gap-4"
                initial={{ opacity: 0, x: 16 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -16 }}
                transition={spring.smooth}
              >
                <p className="text-[13px] text-mid">
                  Sent to <span className="font-semibold text-ink">{phone}</span>
                </p>
                <OTPInput
                  length={6}
                  onComplete={handleStep1}
                  error={error ? ' ' : undefined}
                  disabled={loading}
                  autoFocus
                  onResend={() => onRequestOtp(phone).catch(() => {})}
                />
                <button
                  type="button"
                  onClick={() => { setStep(0); setError(''); setOtp(''); }}
                  className="text-sm text-center text-mid hover:text-ink transition-colors underline-offset-2 hover:underline cursor-pointer"
                >
                  Change phone number
                </button>
              </motion.div>
            )}

            {step === 2 && (
              <motion.form
                key="step-2"
                onSubmit={handleStep2}
                className="flex flex-col gap-4"
                noValidate
                initial={{ opacity: 0, x: 16 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -16 }}
                transition={spring.smooth}
              >
                <PasswordInput
                  id="new-password"
                  label="New password"
                  placeholder="Min. 8 characters"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                  required
                />
                <PasswordInput
                  id="confirm-password"
                  label="Confirm password"
                  placeholder="Repeat your password"
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  autoComplete="new-password"
                  required
                />
                <Button type="submit" variant="primary" size="lg" isLoading={loading} className="w-full">
                  Set new password
                </Button>
              </motion.form>
            )}
          </AnimatePresence>
        </div>
      )}
    </AuthShell>
  );
}
