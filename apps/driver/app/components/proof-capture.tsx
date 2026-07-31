'use client';

import { useRef, useState } from 'react';
import { proofApi, sha256Hex, type ProofKind } from '@speedplus/api-client';

/**
 * ProofCapture is the chain-of-custody media step: the rider captures a live
 * photo/video (camera only — `capture="environment"`, no gallery picker),
 * we hash it, presign an R2 upload, PUT the bytes directly to R2, then record
 * the append-only evidence row. GPS is attached best-effort. A seal serial
 * pairs the pickup and dropoff shots of the same tamper-evident seal.
 */
export function ProofCapture({
  orderId,
  stopId,
  kind,
  label,
  onCaptured,
}: {
  orderId: string;
  stopId?: string;
  kind: ProofKind;
  label: string;
  onCaptured?: (measuredKg?: number) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [sealSerial, setSealSerial] = useState('');
  const [measuredKgInput, setMeasuredKgInput] = useState('');
  const [status, setStatus] = useState<'idle' | 'uploading' | 'done' | 'error'>('idle');
  const [error, setError] = useState('');

  const isVideo = kind.endsWith('_video');
  const isWeightPhoto = kind === 'weight_photo';

  async function currentCoords(): Promise<{ lat?: number; lng?: number }> {
    return new Promise((resolve) => {
      if (!('geolocation' in navigator)) return resolve({});
      navigator.geolocation.getCurrentPosition(
        (p) => resolve({ lat: p.coords.latitude, lng: p.coords.longitude }),
        () => resolve({}),
        { timeout: 4000 },
      );
    });
  }

  async function handleFile(file: File) {
    setStatus('uploading');
    setError('');
    try {
      const contentType = file.type || (isVideo ? 'video/mp4' : 'image/jpeg');
      const measuredKg = isWeightPhoto && measuredKgInput ? parseFloat(measuredKgInput) : undefined;
      const [hash, coords, presigned] = await Promise.all([
        sha256Hex(file),
        currentCoords(),
        proofApi.presign(orderId, { kind, contentType, stopId }),
      ]);
      await proofApi.uploadToPresignedUrl(presigned.uploadUrl, file, contentType);
      await proofApi.confirm(orderId, {
        kind,
        key: presigned.key,
        sha256: hash,
        stopId,
        sealSerial: sealSerial.trim() || undefined,
        measuredKg: measuredKg && !isNaN(measuredKg) ? measuredKg : undefined,
        capturedLat: coords.lat,
        capturedLng: coords.lng,
      });
      setStatus('done');
      onCaptured?.(measuredKg && !isNaN(measuredKg) ? measuredKg : undefined);
    } catch (e) {
      setStatus('error');
      setError(e instanceof Error ? e.message : 'Upload failed');
    }
  }

  return (
    <div className="flex flex-col gap-2 rounded-xl border border-line bg-white/60 p-3">
      <span className="text-[12px] font-semibold text-emerald">{label}</span>

      {isWeightPhoto ? (
        <input
          aria-label="Measured weight in kg"
          type="number"
          inputMode="decimal"
          step="0.1"
          min="0"
          value={measuredKgInput}
          onChange={(e) => setMeasuredKgInput(e.target.value)}
          placeholder="Scale reading (kg) e.g. 12.3"
          className="rounded-lg border border-line px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
        />
      ) : (
        <input
          aria-label="Seal serial number"
          value={sealSerial}
          onChange={(e) => setSealSerial(e.target.value)}
          placeholder="Seal number (on the tamper sticker)"
          className="rounded-lg border border-line px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
        />
      )}

      <input
        ref={inputRef}
        type="file"
        accept={isVideo ? 'video/*' : 'image/*'}
        capture="environment"
        className="hidden"
        onChange={(e) => {
          const f = e.target.files?.[0];
          if (f) void handleFile(f);
        }}
      />

      <button
        type="button"
        disabled={status === 'uploading' || status === 'done'}
        onClick={() => inputRef.current?.click()}
        className="rounded-xl bg-emerald px-4 py-2.5 text-[13px] font-semibold text-sand transition-colors hover:bg-emerald/90 disabled:opacity-50"
      >
        {status === 'uploading'
          ? 'Uploading…'
          : status === 'done'
            ? 'Captured ✓'
            : isVideo
              ? 'Record short video'
              : 'Take photo'}
      </button>

      {error && (
        <span role="alert" className="text-[12px] text-red-600">
          {error}
        </span>
      )}
    </div>
  );
}
