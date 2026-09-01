'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { merchantApi } from '@fourdat/api-client';
import { Card, Badge, Button } from '@fourdat/ui';

function naira(kobo: number) {
  return (kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 });
}

const STAGE: Record<string, {
  label: string;
  variant: 'success' | 'warning' | 'error' | 'default';
  dot: string;
  actLabel: string | null;
  next: string | null;
  doneLabel: string | null;
}> = {
  pending:          { label: 'New',             variant: 'success', dot: '#C6F24E', actLabel: 'Confirm & prepare', next: 'confirmed',        doneLabel: null },
  confirmed:        { label: 'Confirmed',        variant: 'success', dot: '#C6F24E', actLabel: 'Start preparing',   next: 'preparing',         doneLabel: null },
  preparing:        { label: 'Preparing',        variant: 'warning', dot: '#E8B14E', actLabel: 'Mark ready',        next: 'ready_for_pickup',  doneLabel: null },
  ready_for_pickup: { label: 'Awaiting rider',   variant: 'default', dot: '#0A3D2C', actLabel: null,                next: null,                doneLabel: 'Rider on the way' },
  driver_assigned:  { label: 'Rider assigned',   variant: 'default', dot: '#0A3D2C', actLabel: null,                next: null,                doneLabel: 'Rider on the way' },
  in_transit:       { label: 'In transit',       variant: 'default', dot: '#0A3D2C', actLabel: null,                next: null,                doneLabel: 'Out for delivery' },
  delivered:        { label: 'Delivered',        variant: 'default', dot: '#BDBAB2', actLabel: null,                next: null,                doneLabel: 'Completed' },
  cancelled:        { label: 'Cancelled',        variant: 'error',   dot: '#DC2626', actLabel: null,                next: null,                doneLabel: 'Cancelled' },
};

export default function OrdersPage() {
  const qc = useQueryClient();

  const ordersQuery = useQuery({
    queryKey: ['merchant-orders'],
    queryFn: () => merchantApi.listOrders(),
    refetchInterval: 15_000,
  });

  const transitionMutation = useMutation({
    mutationFn: ({ id, to }: { id: string; to: string }) => merchantApi.transitionOrder(id, to),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['merchant-orders'] }),
  });

  const orders = ordersQuery.data?.orders ?? [];

  return (
    <>
      <div>
        <p className="text-xs font-semibold text-mid tracking-widest uppercase">Partner Portal</p>
        <h1 className="font-display font-bold text-2xl text-ink tracking-tight mt-0.5">Orders</h1>
      </div>

      {ordersQuery.isLoading && (
        <p className="text-sm text-mid">Loading orders…</p>
      )}

      {!ordersQuery.isLoading && orders.length === 0 && (
        <Card>
          <p className="text-sm text-mid text-center py-8">No orders yet.</p>
        </Card>
      )}

      <div className="flex flex-col gap-3">
        {orders.map((o) => {
          const meta = STAGE[o.status] ?? STAGE['pending']!;
          const itemSummary = o.items.map((i) => `${i.name} ×${i.quantity}`).join(', ');
          return (
            <Card key={o.id} className="flex items-center gap-4 p-4">
              <span
                className="w-2 h-2 flex-none rounded-full"
                style={{ background: meta.dot }}
              />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold text-ink truncate">{itemSummary || 'Order'}</p>
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
                  variant="primary"
                  onClick={() => transitionMutation.mutate({ id: o.id, to: meta.next! })}
                  disabled={transitionMutation.isPending}
                  isLoading={transitionMutation.isPending}
                >
                  {meta.actLabel}
                </Button>
              ) : (
                <span className="w-36 text-center text-xs text-mid">{meta.doneLabel}</span>
              )}
            </Card>
          );
        })}
      </div>
    </>
  );
}
