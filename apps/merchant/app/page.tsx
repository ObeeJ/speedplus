import { SpeedPlusLogo } from '@speedplus/ui';

export default function MerchantHomePage() {
  return (
    <main className="min-h-screen bg-midnight text-white flex flex-col items-center justify-center gap-6 px-6">
      <SpeedPlusLogo variant="full" theme="dark" size="lg" />
      <h1 className="font-display font-bold text-2xl">Merchant Portal</h1>
      <p className="text-mid text-center max-w-sm">
        Manage your store, handle orders, and grow your business on SpeedPlus.
      </p>
      <a
        href="/login"
        className="mt-4 rounded-xl bg-primary px-8 py-3 font-semibold text-midnight hover:bg-primary-700 transition-colors"
      >
        Open Dashboard
      </a>
    </main>
  );
}
