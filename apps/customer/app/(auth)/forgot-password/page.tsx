'use client';

import { ForgotPasswordFlow } from '@speedplus/ui';
import { authApi } from '@speedplus/api-client';
import { useRouter } from 'next/navigation';

export default function ForgotPasswordPage() {
  const router = useRouter();

  return (
    <ForgotPasswordFlow
      headline={<>Faster.<br />Cheaper.<br /><span className="text-lime">Better.</span></>}
      subtext="Essential delivery for everyday Nigeria."
      onRequestOtp={(phone) => authApi.requestOtp(phone, 'password_reset')}
      onVerifyOtp={(phone, otp) => authApi.verifyOtpCode(phone, otp, 'password_reset').then(() => {})}
      onResetPassword={(phone, otp, newPassword) => authApi.resetPassword({ phone, otp, newPassword })}
      loginHref="/login"
    />
  );
}
