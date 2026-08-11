'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { merchantApi } from '@speedplus/api-client';
import { Card, CardHeader, CardTitle, CardContent, Badge, Button, Progress, StatCard, StatCardSkeleton } from '@speedplus/ui';
import { useMerchantAuthStore } from '@/lib/store/auth.store';

function naira(kobo: number) {
  return (kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 });
}

export default function DashboardPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const { merchant } = useMerchantAuthStore();

  const profileQuery = useQuery({
    queryKey: ['merchant-profile'],
    queryFn: () => merchantApi.getProfile(),
    initialData: merchant ?? undefined,
  });

  const ordersQuery = useQuery({
    queryKey: ['merchant-orders'],
    queryFn: () => merchantApi.listOrders(),
    refetchInterval: 15_000,
  });

  const productsQuery = useQuery({
    queryKey: ['merchant-products'],
    queryFn: () => merchantApi.listProducts(),
  });

  const toggleOpenMutation = useMutation({
    mutationFn: (isOpen: boolean) => merchantApi.setOpen(isOpen),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['merchant-profile'] }),
  });

  const orders = ordersQuery.data?.orders ?? [];
  const products = productsQuery.data?.products ?? [];
  const newCount = orders.filter((o) => o.status === 'pending' || o.status === 'confirmed').length;
  const todaySalesKobo = orders
    .filter((o) => o.status === 'delivered')
    .reduce((sum, o) => sum + o.total.amount, 0);
  const availableProducts = products.filter((p) => p.isAvailable).length;

  const profile = profileQuery.data;

  return (
    <>
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs font-semibold text-mid tracking-widest uppercase">Partner Portal</p>
          <h1 className="font-display font-bold text-2xl text-ink tracking-tight mt-0.5">
            Good day, {profile?.businessName ?? '…'}
          </h1>
        </div>
        {profile && (
          <Button
            variant={profile.isOpen ? 'outline' : 'primary'}
            size="sm"
            onClick={() => toggleOpenMutation.mutate(!profile.isOpen)}
            disabled={toggleOpenMutation.isPending}
          >
            <span className={`w-1.5 h-1.5 rounded-full ${profile.isOpen ? 'bg-emerald animate-pulse' : 'bg-mid'}`} />
            {profile.isOpen ? 'Open — tap to close' : 'Closed — tap to open'}
          </Button>
        )}
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {ordersQuery.isLoading || productsQuery.isLoading ? (
          <>
            <StatCardSkeleton /><StatCardSkeleton /><StatCardSkeleton /><StatCardSkeleton />
          </>
        ) : (
          <>
            <StatCard label="Delivered today" value={`₦${naira(todaySalesKobo)}`} />
            <StatCard label="Orders" value={String(orders.length)} sub={`${newCount} need action`} />
            <StatCard
              label="Products"
              value={String(products.length)}
              sub={`${availableProducts} listed`}
            />
            <StatCard
              label="Rating"
              value={`★ ${profile?.rating.toFixed(1) ?? '—'}`}
              sub={profile?.kycStatus === 'approved' ? 'Verified' : (profile?.kycStatus ?? '—')}
            />
          </>
        )}
      </div>

      {/* Gas-only cards */}
      {profile?.vertical === 'gas' && (
        <div className="grid grid-cols-2 gap-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-xs font-semibold text-mid tracking-widest uppercase">
                Fill accuracy
              </CardTitle>
            </CardHeader>
            <CardContent>
              {profile.fillAccuracyPct != null ? (
                <>
                  <p
                    className={[
                      'font-display text-2xl font-bold',
                      profile.fillAccuracyPct >= 0.98 ? 'text-emerald'
                        : profile.fillAccuracyPct >= 0.95 ? 'text-amber'
                        : 'text-red-700',
                    ].join(' ')}
                  >
                    {(profile.fillAccuracyPct * 100).toFixed(1)}%
                  </p>
                  <p className="text-xs text-mid mt-1">{profile.fillSampleCount ?? 0} verified fills</p>
                  <Progress
                    value={profile.fillAccuracyPct * 100}
                    variant={profile.fillAccuracyPct >= 0.95 ? 'lime' : 'emerald'}
                    className="mt-2"
                  />
                </>
              ) : (
                <>
                  <p className="font-display text-2xl font-bold text-mid">—</p>
                  <p className="text-xs text-mid mt-1">No verified fills yet</p>
                </>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-xs font-semibold text-mid tracking-widest uppercase">
                Gas orders today
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="font-display text-2xl font-bold text-ink">
                {orders.filter((o) => o.vertical === 'gas').length}
              </p>
              <p className="text-xs text-mid mt-1">
                {orders.filter((o) => o.vertical === 'gas' && o.status === 'delivered').length} delivered
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Action banner */}
      {newCount > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-xs font-semibold text-mid tracking-widest uppercase">
              Needs your action now
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-3 bg-tile rounded-xl px-4 py-3">
              <span className="w-2 h-2 rounded-full bg-emerald flex-none" />
              <p className="flex-1 text-sm font-semibold text-ink">
                {newCount} order{newCount > 1 ? 's' : ''} to confirm
              </p>
              <Button size="sm" variant="outline" onClick={() => router.push('/orders')}>
                View orders
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </>
  );
}
