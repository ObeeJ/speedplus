'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function GasFindingPage() {
  const router = useRouter();

  useEffect(() => {
    const timer = setTimeout(() => router.push('/gas/tracking'), 2200);
    return () => clearTimeout(timer);
  }, [router]);

  return (
    <main className="min-h-screen bg-emerald flex flex-col items-center justify-center gap-5 px-5">
      <span className="w-16 h-16 rounded-full border-4 border-sand/20 border-t-lime animate-spin" />
      <span className="font-display font-semibold text-xl text-sand text-center">Finding you a rider…</span>
      <span className="text-sm text-sand/60 text-center">This usually takes a few seconds</span>
    </main>
  );
}
