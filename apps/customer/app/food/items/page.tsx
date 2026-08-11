'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Button, Skeleton } from '@speedplus/ui';
import { catalogApi } from '@speedplus/api-client';
import { FlowHeader } from '../../components/flow-header';
import { useFoodFlowStore } from '../../../lib/store/food-flow.store';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

export default function FoodItemsPage() {
  const router = useRouter();
  const { merchantId, productId, setProduct } = useFoodFlowStore();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['food-products', merchantId],
    queryFn: () => catalogApi.listProducts(merchantId!),
    enabled: Boolean(merchantId),
  });

  if (!merchantId) {
    router.replace('/food/menu');
    return null;
  }

  const available = data?.products.filter((p) => p.isAvailable) ?? [];

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Pick your meal" step={2} totalSteps={4} backHref="/food/menu" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-4 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        {isLoading && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-[62px] rounded-[13px]" />
            <Skeleton className="h-[62px] rounded-[13px]" />
            <Skeleton className="h-[62px] rounded-[13px]" />
          </div>
        )}

        {isError && (
          <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">
            Couldn&apos;t load the menu. Please try again.
          </div>
        )}

        {!isLoading && available.length === 0 && (
          <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">
            Nothing available right now. Try another kitchen.
          </div>
        )}

        <div className="flex flex-col gap-2">
          {available.map((product) => {
            const selected = productId === product.id;
            return (
              <button
                key={product.id}
                data-testid="product-add-btn"
                onClick={() => setProduct(product.id, product.priceKobo)}
                className={`w-full flex items-center justify-between gap-3 rounded-[13px] border-2 px-4 py-3.5 text-left transition-all ${
                  selected ? 'bg-emerald border-lime' : 'bg-white border-line hover:border-emerald/40'
                }`}
              >
                <span className="flex flex-col gap-0.5">
                  <span className={`font-display font-semibold text-[15px] ${selected ? 'text-lime' : 'text-ink'}`}>
                    {selected ? `✓ ${product.name}` : product.name}
                  </span>
                  {product.description && (
                    <span className={`text-[12.5px] ${selected ? 'text-sand/70' : 'text-mid'}`}>{product.description}</span>
                  )}
                </span>
                <span className={`font-display font-semibold text-[14px] ${selected ? 'text-lime' : 'text-ink'}`}>
                  {naira(product.priceKobo)}
                </span>
              </button>
            );
          })}
        </div>

        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!productId}
          onClick={() => router.push('/food/deliver')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>
    </main>
  );
}
