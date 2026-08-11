'use client';

import { useState } from 'react';
import { adminApi } from '@speedplus/api-client';

export default function LPGPricePage() {
  const [region, setRegion] = useState('Lagos');
  const [price, setPrice] = useState('');
  const [source, setSource] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ pricePerKgKobo: number; suggestion: string | null } | null>(null);
  const [error, setError] = useState('');

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const priceKobo = Math.round(parseFloat(price) * 100);
    if (!region.trim() || !priceKobo || !source.trim()) {
      setError('All fields are required. Price is in naira (e.g. 1200).');
      return;
    }
    setLoading(true);
    setError('');
    setResult(null);
    try {
      const res = await adminApi.recordLPGPrice({ region: region.trim(), pricePerKgKobo: priceKobo, source: source.trim() });
      setResult({ pricePerKgKobo: priceKobo, suggestion: res.suggestion as string | null });
      setPrice('');
      setSource('');
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to record price.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="max-w-lg">
      <h1 className="text-xl font-semibold mb-1">LPG Price Index</h1>
      <p className="text-sm text-gray-500 mb-6">Record the current market price per kg. A &gt;10% move generates a product-price suggestion but never auto-applies it.</p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium">Region</label>
          <input
            value={region}
            onChange={(e) => setRegion(e.target.value)}
            placeholder="e.g. Lagos"
            className="border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500" aria-label="Region"/>
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium">Price per kg (₦)</label>
          <input
            type="number"
            step="0.01"
            min="1"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            placeholder="e.g. 1200"
            className="border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500" aria-label="Price per kg (₦)"/>
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium">Source</label>
          <input
            value={source}
            onChange={(e) => setSource(e.target.value)}
            placeholder="e.g. NMDPRA bulletin, manual survey"
            className="border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500" aria-label="Source"/>
        </div>

        {error && <p className="text-sm text-red-600" role="alert">{error}</p>}

        {result && (
          <div className="rounded-lg bg-green-50 border border-green-200 px-4 py-3 text-sm">
            <p className="font-medium text-green-800">Recorded — ₦{(result.pricePerKgKobo / 100).toLocaleString('en-NG')}/kg</p>
            {result.suggestion && (
              <p className="mt-1 text-green-700">Suggestion: {result.suggestion}</p>
            )}
          </div>
        )}

        <button
          type="submit"
          disabled={loading}
          className="bg-emerald-700 hover:bg-emerald-800 disabled:opacity-50 text-white font-semibold rounded-lg px-4 py-2.5 text-sm transition-colors"
        >
          {loading ? 'Recording…' : 'Record price'}
        </button>
      </form>
    </div>
  );
}
