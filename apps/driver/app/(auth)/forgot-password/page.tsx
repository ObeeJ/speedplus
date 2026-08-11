'use client';

import { ForgotPasswordFlow } from '@speedplus/ui';
import { authApi } from '@speedplus/api-client';

export default function ForgotPasswordPage() {
  return (
    <ForgotPasswordFlow
      headline={<>Deliver.<br />Earn.<br /><span className="text-lime">Grow.</span></>}
      subtext="Your earnings, your schedule, your city."
      portalLabel="Rider Portal"
      onRequestOtp={(phone) => authApi.requestOtp(phone, 'password_reset')}
      onVerifyOtp={(phone, otp) => authApi.verifyOtpCode(phone, otp, 'password_reset').then(() => {})}
      onResetPassword={(phone, otp, newPassword) => authApi.resetPassword({ phone, otp, newPassword })}
      loginHref="/login"
    />
  );
}
