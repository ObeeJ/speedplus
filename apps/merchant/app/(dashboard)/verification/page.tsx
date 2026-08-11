'use client';

import { useQuery } from '@tanstack/react-query';
import { merchantApi } from '@speedplus/api-client';
import { Card, Badge } from '@speedplus/ui';
import { ShieldCheckIcon } from '@speedplus/ui';
import { useMerchantAuthStore } from '@/lib/store/auth.store';

export default function VerificationPage() {
  const { merchant } = useMerchantAuthStore();

  const profileQuery = useQuery({
    queryKey: ['merchant-profile'],
    queryFn: () => merchantApi.getProfile(),
    initialData: merchant ?? undefined,
  });

  const profile = profileQuery.data;

  return (
    <>
      <div>
        <p className="text-xs font-semibold text-mid tracking-widest uppercase">Partner Portal</p>
        <h1 className="font-display font-bold text-2xl text-ink tracking-tight mt-0.5">
          Verification &amp; onboarding
        </h1>
        <p className="text-sm text-mid mt-1 max-w-prose">
          Your customers see this status — trust is the product.
        </p>
      </div>

      <Card className="flex items-center gap-4 p-5 max-w-2xl">
        <ShieldCheckIcon size={24} />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-ink">{profile?.businessName}</p>
          <p className="text-xs text-mid mt-0.5 capitalize">Vertical: {profile?.vertical}</p>
        </div>
        <Badge
          variant={profile?.kycStatus === 'approved' ? 'success' : 'warning'}
          className="capitalize"
        >
          {profile?.kycStatus === 'approved'
            ? '✓ Verified'
            : (profile?.kycStatus ?? '—').replace('_', ' ')}
        </Badge>
      </Card>
    </>
  );
}
