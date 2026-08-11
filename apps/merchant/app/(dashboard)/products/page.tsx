'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { merchantApi, type ProductInput } from '@speedplus/api-client';
import { Card, Button, Input } from '@speedplus/ui';

function naira(kobo: number) {
  return (kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 });
}

const EMPTY: ProductInput = { name: '', description: undefined, priceKobo: 0, category: '', isAvailable: true };

export default function ProductsPage() {
  const qc = useQueryClient();
  const [showAdd, setShowAdd] = useState(false);
  const [draft, setDraft] = useState<ProductInput>(EMPTY);

  const productsQuery = useQuery({
    queryKey: ['merchant-products'],
    queryFn: () => merchantApi.listProducts(),
  });

  const createMutation = useMutation({
    mutationFn: (input: ProductInput) => merchantApi.createProduct(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['merchant-products'] });
      setShowAdd(false);
      setDraft(EMPTY);
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, available }: { id: string; available: boolean }) =>
      merchantApi.setProductAvailability(id, available),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['merchant-products'] }),
  });

  const products = productsQuery.data?.products ?? [];

  return (
    <>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs font-semibold text-mid tracking-widest uppercase">Partner Portal</p>
          <h1 className="font-display font-bold text-2xl text-ink tracking-tight mt-0.5">Products</h1>
        </div>
        <Button size="sm" onClick={() => setShowAdd((v) => !v)}>
          + Add product
        </Button>
      </div>

      {showAdd && (
        <Card>
          <p className="text-xs font-semibold text-mid tracking-widest uppercase mb-4">New product</p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Input
              id="prod-name"
              label="Product name"
              placeholder="e.g. Paracetamol 500mg"
              value={draft.name}
              onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
            />
            <Input
              id="prod-category"
              label="Category"
              placeholder="e.g. Analgesics"
              value={draft.category}
              onChange={(e) => setDraft((d) => ({ ...d, category: e.target.value }))}
            />
            <Input
              id="prod-price"
              label="Price (₦)"
              type="number"
              placeholder="0"
              value={draft.priceKobo ? String(draft.priceKobo / 100) : ''}
              onChange={(e) =>
                setDraft((d) => ({ ...d, priceKobo: Math.round(Number(e.target.value) * 100) }))
              }
            />
          </div>
          <div className="flex gap-3 mt-4">
            <Button
              size="sm"
              onClick={() => createMutation.mutate(draft)}
              disabled={!draft.name || !draft.priceKobo}
              isLoading={createMutation.isPending}
            >
              Save product
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setShowAdd(false)}>
              Cancel
            </Button>
          </div>
        </Card>
      )}

      {productsQuery.isLoading && <p className="text-sm text-mid">Loading products…</p>}

      {!productsQuery.isLoading && products.length === 0 && (
        <Card>
          <p className="text-sm text-mid text-center py-8">No products yet — add your first one.</p>
        </Card>
      )}

      <div className="flex flex-col gap-3">
        {products.map((p) => (
          <Card key={p.id} className="flex items-center gap-4 p-4">
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold text-ink">{p.name}</p>
              <p className="text-xs text-mid mt-0.5">{p.category || 'Uncategorized'}</p>
            </div>
            <p className="font-display text-sm font-bold text-ink w-20 text-right">
              ₦{naira(p.priceKobo)}
            </p>
            {/* Toggle switch */}
            <button
              onClick={() => toggleMutation.mutate({ id: p.id, available: !p.isAvailable })}
              disabled={toggleMutation.isPending}
              className="w-11 h-6 flex-none rounded-full relative transition-colors disabled:opacity-50"
              style={{ background: p.isAvailable ? '#0A3D2C' : '#D5D2C8' }}
              aria-label={p.isAvailable ? 'Disable product' : 'Enable product'}
              role="switch"
              aria-checked={p.isAvailable}
            >
              <span
                className="absolute top-[3px] w-[18px] h-[18px] rounded-full bg-white shadow-sm transition-all duration-150"
                style={{ left: p.isAvailable ? 23 : 3 }}
              />
            </button>
          </Card>
        ))}
      </div>
    </>
  );
}
