'use client';

import { ForgotPasswordFlow } from '@fourdat/ui';
import { authApi } from '@fourdat/api-client';

export default function ForgotPasswordPage() {
  return (
    <ForgotPasswordFlow
      headline={<>Your business,<br /><span className="text-lime">amplified.</span></>}
      subtext="Manage orders, products, prescriptions, and earnings — all in one place."
      portalLabel="Partner Portal"
      onRequestOtp={(phone) => authApi.requestOtp(phone, 'password_reset')}
      onVerifyOtp={(phone, otp) => authApi.verifyOtpCode(phone, otp, 'password_reset').then(() => {})}
      onResetPassword={(phone, otp, newPassword) => authApi.resetPassword({ phone, otp, newPassword })}
      loginHref="/login"
    />
  );
}
