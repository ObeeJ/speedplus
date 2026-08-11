'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { merchantApi } from '@speedplus/api-client';
import { Card, CardHeader, CardTitle, CardContent, Badge, Button, Progress } from '@speedplus/ui';
import { GasIcon } from '@speedplus/ui';
import { useMerchantAuthStore } from '@/lib/store/auth.store';

function naira(kobo: number) {
  return (kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 });
}

const GAS_STAGE: Record<string, {
  label: string;
  variant: 'success' | 'warning' | 'error' | 'default';
  actLabel: string | null;
  next: string | null;
}> = {
  pending:          { label: 'New',             variant: 'success', actLabel: 'Confirm order',    next: 'confirmed' },
  confirmed:        { label: 'Confirmed',        variant: 'success', actLabel: 'Ready for rider',  next: 'ready_for_pickup' },
  preparing:        { label: 'Preparing',        variant: 'warning', actLabel: 'Ready for rider',  next: 'ready_for_pickup' },
  ready_for_pickup: { label: 'Awaiting rider',   variant: 'default', actLabel: null,               next: null },
  driver_assigned:  { label: 'Rider assigned',   variant: 'default', actLabel: null,               next: null },
  in_transit:       { label: 'Out for delivery', variant: 'default', actLabel: null,               next: null },
  delivered:        { label: 'Delivered',        variant: 'default', actLabel: null,               next: null },
  cancelled:        { label: 'Cancelled',        variant: 'error',   actLabel: null,               next: null },
};

export default function GasOpsPage() {
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

  const transitionMutation = useMutation({
    mutationFn: ({ id, to }: { id: string; to: string }) => merchantApi.transitionOrder(id, to),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['merchant-orders'] }),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, available }: { id: string; available: boolean }) =>
      merchantApi.setProductAvailability(id, available),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['merchant-products'] }),
  });

  const profile = profileQuery.data;
  const allOrders = ordersQuery.data?.orders ?? [];
  const gasOrders = allOrders.filter((o) => o.vertical === 'gas');
  const cylinderProducts = (productsQuery.data?.products ?? []).filter((p) =>
    p.name.toLowerCase().includes('kg'),
  );

  const accuracyPct = profile?.fillAccuracyPct;
  const accuracyColor =
    accuracyPct == null ? '#63636E'
    : accuracyPct >= 0.98 ? '#0A3D2C'
    : accuracyPct >= 0.95 ? '#8A6A1B'
    : '#B4231F';

  return (
    <>
      <div>
        <p className="text-xs font-semibold text-mid tracking-widest uppercase">Partner Portal</p>
        <h1 className="font-display font-bold text-2xl text-ink tracking-tight mt-0.5">Gas operations</h1>
      </div>

      {/* Fill accuracy */}
      <Card>
        <CardHeader>
          <CardTitle className="text-xs font-semibold text-mid tracking-widest uppercase">
            Fill accuracy score
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-start gap-8">
            <div className="flex flex-col gap-1">
              <p
                className="font-display text-5xl font-bold leading-none"
                style={{ color: accuracyColor }}
              >
                {accuracyPct != null ? `${(accuracyPct * 100).toFixed(1)}%` : '—'}
              </p>
              <p className="text-xs text-mid mt-1">
                {profile?.fillSampleCount ?? 0} verified fills
              </p>
              {accuracyPct != null && (
                <Progress
                  value={accuracyPct * 100}
                  variant={accuracyPct >= 0.95 ? 'lime' : 'emerald'}
                  className="mt-2 w-32"
                />
              )}
            </div>
            <div className="flex-1 flex flex-col gap-1.5 text-sm text-mid max-w-sm">
              <p className="font-semibold text-ink text-sm">How this works</p>
              <p>Every gas delivery is weighed at the customer&apos;s door. The scale photo is recorded and ordered vs measured weight is compared automatically.</p>
              <p>Short by more than 2%? The difference is refunded from your settlement before the rider leaves.</p>
              {accuracyPct != null && (
                <p
                  className="font-semibold text-sm mt-1"
                  style={{ color: accuracyPct < 0.95 ? '#B4231F' : '#0A3D2C' }}
                >
                  {accuracyPct < 0.95
                    ? '⚠️ Below 95% — risk of delisting. Improve fill accuracy to stay on the platform.'
                    : '✓ Good standing'}
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Cylinder float stock */}
      <div>
        <p className="text-xs font-semibold text-mid tracking-widest uppercase mb-3">
          Cylinder float stock
        </p>
        {cylinderProducts.length === 0 ? (
          <Card>
            <p className="text-sm text-mid text-center py-6">
              No cylinder products found. Add them in the Products tab.
            </p>
          </Card>
        ) : (
          <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))' }}>
            {cylinderProducts.map((p) => (
              <Card key={p.id} className="flex flex-col gap-3">
                <div>
                  <p className="text-xs font-semibold text-mid tracking-widest uppercase">
                    {p.name
                      .replace(' LPG cylinder', '')
                      .replace(' cylinder', '')
                      .toUpperCase()}
                  </p>
                  <p className="text-sm font-bold text-ink mt-1">₦{naira(p.priceKobo)}</p>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className="w-2 h-2 rounded-full flex-none"
                    style={{ background: p.isAvailable ? '#C6F24E' : '#D5D2C8' }}
                  />
                  <span className="text-xs font-semibold text-ink">
                    {p.isAvailable ? 'In stock' : 'Out of stock'}
                  </span>
                </div>
                <Button
                  size="sm"
                  variant={p.isAvailable ? 'danger' : 'outline'}
                  onClick={() => toggleMutation.mutate({ id: p.id, available: !p.isAvailable })}
                  disabled={toggleMutation.isPending}
                >
                  {p.isAvailable ? 'Mark out of stock' : 'Mark in stock'}
                </Button>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* Gas order queue */}
      <div>
        <p className="text-xs font-semibold text-mid tracking-widest uppercase mb-3">
          Gas order queue
        </p>

        {ordersQuery.isLoading && <p className="text-sm text-mid">Loading…</p>}

        {!ordersQuery.isLoading && gasOrders.length === 0 && (
          <Card>
            <p className="text-sm text-mid text-center py-6">No gas orders yet.</p>
          </Card>
        )}

        <div className="flex flex-col gap-3">
          {gasOrders.map((o) => {
            const meta = GAS_STAGE[o.status] ?? GAS_STAGE['pending']!;
            const cylinderName = o.items[0]?.name ?? 'Gas cylinder';
            return (
              <Card key={o.id} className="flex items-center gap-4 p-4">
                <GasIcon
                  size={18}
                  active={false}
                  color="#1C3A2E"
                  accent="#7BA05B"
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-ink truncate">{cylinderName}</p>
                  <p className="text-xs text-mid mt-0.5">
                    #{o.id.slice(0, 8)} · {o.paymentMethod}
                  </p>
                </div>
                <Badge variant={meta.variant}>{meta.label}</Badge>
                <p className="font-display text-base font-bold text-emerald w-24 text-right">
                  ₦{naira(o.total.amount)}
                </p>
                {meta.actLabel && meta.next ? (
                  <Button
                    size="sm"
                    onClick={() => transitionMutation.mutate({ id: o.id, to: meta.next! })}
                    disabled={transitionMutation.isPending}
                    isLoading={transitionMutation.isPending}
                  >
                    {meta.actLabel}
                  </Button>
                ) : (
                  <span className="w-36 text-center text-xs text-mid capitalize">
                    {meta.label.toLowerCase()}
                  </span>
                )}
              </Card>
            );
          })}
        </div>
      </div>
    </>
  );
}
