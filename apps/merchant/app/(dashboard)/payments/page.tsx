'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { paycodesApi } from '@speedplus/api-client';
import { Card, Button, Input } from '@speedplus/ui';

export default function PaymentsPage() {
  const [paycodeOrderId, setPaycodeOrderId] = useState('');
  const [generatedPaycode, setGeneratedPaycode] = useState<{ id: string; payload: string } | null>(null);
  const [scanPayload, setScanPayload] = useState('');
  const [scanPin, setScanPin] = useState('');
  const [scanResult, setScanResult] = useState<string | null>(null);

  const generateMutation = useMutation({
    mutationFn: (orderId: string) => paycodesApi.generate(orderId),
    onSuccess: (data) => setGeneratedPaycode({ id: data.id, payload: data.payload }),
  });

  const scanMutation = useMutation({
    mutationFn: () => paycodesApi.scanCard(scanPayload.trim(), scanPin),
    onSuccess: () => { setScanResult('Payment confirmed'); setScanPayload(''); setScanPin(''); },
    onError: (e: Error) => setScanResult(e.message),
  });

  return (
    <>
      <div>
        <p className="text-xs font-semibold text-mid tracking-widest uppercase">Partner Portal</p>
        <h1 className="font-display font-bold text-2xl text-ink tracking-tight mt-0.5">Payments</h1>
      </div>

      {/* Generate paycode */}
      <Card className="flex flex-col gap-4">
        <p className="text-xs font-semibold text-mid tracking-widest uppercase">Generate paycode</p>
        <p className="text-sm text-mid -mt-2">
          Enter an order ID to generate a 6-digit delivery code for the rider.
        </p>
        <Input
          id="paycode-order"
          label="Order ID"
          placeholder="Order UUID"
          value={paycodeOrderId}
          onChange={(e) => { setPaycodeOrderId(e.target.value); setGeneratedPaycode(null); }}
        />
        {generateMutation.isError && (
          <p className="text-xs text-red-600">{(generateMutation.error as Error).message}</p>
        )}
        {generatedPaycode && (
          <div className="bg-tile rounded-xl px-4 py-3 flex flex-col items-center gap-1">
            <p className="text-[10px] font-semibold text-mid uppercase tracking-widest">Paycode payload</p>
            <p className="font-mono text-sm font-bold text-emerald break-all text-center">
              {generatedPaycode.payload}
            </p>
          </div>
        )}
        <Button
          onClick={() => { if (paycodeOrderId.trim()) generateMutation.mutate(paycodeOrderId.trim()); }}
          disabled={!paycodeOrderId.trim()}
          isLoading={generateMutation.isPending}
        >
          Generate
        </Button>
      </Card>

      {/* Scan card */}
      <Card className="flex flex-col gap-4">
        <p className="text-xs font-semibold text-mid tracking-widest uppercase">Scan SpeedPlus card</p>
        <p className="text-sm text-mid -mt-2">
          Scan a customer&apos;s SpeedPlus card QR payload to confirm payment.
        </p>
        <Input
          id="scan-payload"
          label="Card QR payload"
          placeholder="Scan or paste QR payload"
          value={scanPayload}
          onChange={(e) => { setScanPayload(e.target.value); setScanResult(null); }}
        />
        <Input
          id="scan-pin"
          label="Customer PIN"
          type="password"
          inputMode="numeric"
          maxLength={6}
          placeholder="••••••"
          value={scanPin}
          onChange={(e) => setScanPin(e.target.value.replace(/\D/g, ''))}
        />
        {scanResult && (
          <p
            className={`text-sm font-semibold ${scanResult === 'Payment confirmed' ? 'text-emerald' : 'text-red-600'}`}
            role="status"
          >
            {scanResult === 'Payment confirmed' ? '✓ ' : '✗ '}{scanResult}
          </p>
        )}
        <Button
          variant="secondary"
          onClick={() => scanMutation.mutate()}
          disabled={!scanPayload.trim() || scanPin.length < 4}
          isLoading={scanMutation.isPending}
        >
          Confirm payment
        </Button>
      </Card>
    </>
  );
}
