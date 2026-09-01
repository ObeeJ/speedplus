'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { merchantApi } from '@fourdat/api-client';
import { Card, CardHeader, CardTitle, CardContent, Button, Input, Modal } from '@fourdat/ui';

function naira(kobo: number) {
  return (kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 });
}

export default function EarningsPage() {
  const qc = useQueryClient();
  const [showBankForm, setShowBankForm] = useState(false);
  const [bankDraft, setBankDraft] = useState({ bankCode: '', bankName: '', accountNumber: '', accountName: '' });
  const [showWithdraw, setShowWithdraw] = useState(false);
  const [withdrawAmount, setWithdrawAmount] = useState('');
  const [withdrawPin, setWithdrawPin] = useState('');
  const [withdrawType, setWithdrawType] = useState<'standard' | 'instant'>('standard');

  const walletQuery = useQuery({ queryKey: ['merchant-wallet'], queryFn: () => merchantApi.getWallet() });
  const bankQuery = useQuery({ queryKey: ['merchant-bank-account'], queryFn: () => merchantApi.getBankAccount() });
  const txQuery = useQuery({ queryKey: ['merchant-transactions'], queryFn: () => merchantApi.getTransactions() });

  const saveBankMutation = useMutation({
    mutationFn: () => merchantApi.saveBankAccount(bankDraft),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['merchant-bank-account'] }); setShowBankForm(false); },
  });

  const withdrawMutation = useMutation({
    mutationFn: () => merchantApi.withdraw(
      Math.round(Number(withdrawAmount) * 100),
      withdrawPin,
      crypto.randomUUID(),
      withdrawType,
    ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['merchant-wallet'] });
      qc.invalidateQueries({ queryKey: ['merchant-transactions'] });
      setShowWithdraw(false);
      setWithdrawAmount('');
      setWithdrawPin('');
    },
  });

  const txList = (txQuery.data?.transactions ?? []) as { id: string; description: string; amountKobo: number; createdAt: string }[];

  return (
    <>
      <div>
        <p className="text-xs font-semibold text-mid tracking-widest uppercase">Partner Portal</p>
        <h1 className="font-display font-bold text-2xl text-ink tracking-tight mt-0.5">Earnings</h1>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {/* Wallet card */}
        <div className="rounded-[16px] bg-emerald p-5 flex flex-col gap-2">
          <p className="text-xs font-semibold text-sand/60 tracking-widest uppercase">Wallet balance</p>
          <p className="font-display text-3xl font-bold text-lime">
            {walletQuery.isLoading ? '…' : `₦${naira(walletQuery.data?.balanceKobo ?? 0)}`}
          </p>
          <Button
            size="sm"
            variant="primary"
            className="self-start mt-1"
            onClick={() => setShowWithdraw(true)}
          >
            Withdraw
          </Button>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-xs font-semibold text-mid tracking-widest uppercase">
              Platform commission
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="font-display text-3xl font-bold text-ink">8%</p>
            <p className="text-xs text-mid mt-1">flat, no hidden charges</p>
          </CardContent>
        </Card>
      </div>

      {/* Bank account */}
      <Card className="flex items-center gap-4 p-4">
        {bankQuery.data ? (
          <>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold text-ink">{bankQuery.data.accountName}</p>
              <p className="text-xs text-mid mt-0.5">
                {bankQuery.data.bankName} · {bankQuery.data.accountNumber}
              </p>
            </div>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => { setBankDraft(bankQuery.data!); setShowBankForm(true); }}
            >
              Edit
            </Button>
          </>
        ) : (
          <>
            <p className="flex-1 text-sm text-mid">No bank account linked — add one to withdraw</p>
            <Button size="sm" variant="outline" onClick={() => setShowBankForm(true)}>
              Add account
            </Button>
          </>
        )}
      </Card>

      {showBankForm && (
        <Card>
          <p className="text-xs font-semibold text-mid tracking-widest uppercase mb-4">Bank account details</p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {(['bankCode', 'bankName', 'accountNumber', 'accountName'] as const).map((f) => (
              <Input
                key={f}
                id={`bank-${f}`}
                label={{ bankCode: 'Bank code', bankName: 'Bank name', accountNumber: 'Account number', accountName: 'Account name' }[f]}
                value={bankDraft[f]}
                onChange={(e) => setBankDraft((d) => ({ ...d, [f]: e.target.value }))}
              />
            ))}
          </div>
          <div className="flex gap-3 mt-4">
            <Button
              size="sm"
              onClick={() => saveBankMutation.mutate()}
              disabled={!bankDraft.bankCode || !bankDraft.accountNumber}
              isLoading={saveBankMutation.isPending}
            >
              Save
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setShowBankForm(false)}>Cancel</Button>
          </div>
        </Card>
      )}

      {/* Withdraw modal */}
      <Modal
        isOpen={showWithdraw}
        onClose={() => setShowWithdraw(false)}
        title="Withdraw funds"
        className="max-w-sm"
      >
        <div className="flex flex-col gap-4">
          {!bankQuery.data ? (
              <p className="text-sm text-mid">Add a bank account first before withdrawing.</p>
            ) : (
              <>
                <div>
                  <p className="text-xs font-semibold text-mid tracking-widest uppercase mb-1">To</p>
                  <p className="text-sm font-semibold text-ink">{bankQuery.data.accountName}</p>
                  <p className="text-xs text-mid">{bankQuery.data.bankName} · {bankQuery.data.accountNumber}</p>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  {(['standard', 'instant'] as const).map((type) => (
                    <button
                      key={type}
                      onClick={() => setWithdrawType(type)}
                      className={`rounded-xl border-[1.5px] px-3 py-2.5 text-left transition-colors ${
                        withdrawType === type ? 'border-emerald bg-tile' : 'border-line'
                      }`}
                    >
                      <p className="text-xs font-semibold text-ink capitalize">{type}</p>
                      <p className="text-[11px] text-mid mt-0.5">
                        {type === 'standard' ? 'Free · next business day' : '1% fee · within minutes'}
                      </p>
                    </button>
                  ))}
                </div>
                <Input
                  id="withdraw-amount"
                  label="Amount (₦)"
                  type="number"
                  placeholder="0"
                  value={withdrawAmount}
                  onChange={(e) => setWithdrawAmount(e.target.value)}
                />
                {withdrawType === 'instant' && Number(withdrawAmount) > 0 && (() => {
                  const amtKobo = Math.round(Number(withdrawAmount) * 100);
                  const fee = Math.min(Math.max(Math.round(amtKobo * 0.01), 1000), 50000);
                  return (
                    <p className="text-xs text-mid -mt-2">
                      Fee: ₦{(fee / 100).toLocaleString('en-NG')} · You receive: ₦{((amtKobo - fee) / 100).toLocaleString('en-NG')}
                    </p>
                  );
                })()}
                <Input
                  id="withdraw-pin"
                  label="Wallet PIN"
                  type="password"
                  placeholder="••••••"
                  maxLength={6}
                  value={withdrawPin}
                  onChange={(e) => setWithdrawPin(e.target.value)}
                />
                {withdrawMutation.isError && (
                  <p className="text-xs text-red-600">{(withdrawMutation.error as Error).message}</p>
                )}
              </>
            )}
            <div className="flex gap-3">
              {bankQuery.data && (
                <Button
                  className="flex-1"
                  onClick={() => withdrawMutation.mutate()}
                  disabled={!withdrawAmount || !withdrawPin}
                  isLoading={withdrawMutation.isPending}
                >
                  Confirm
                </Button>
              )}
              <Button className="flex-1" variant="outline" onClick={() => setShowWithdraw(false)}>
                Cancel
              </Button>
            </div>
        </div>
      </Modal>

      {/* Transactions */}
      <Card className="overflow-hidden p-0">
        <div className="flex justify-between px-5 py-3 border-b border-line">
          <p className="text-xs font-semibold text-mid tracking-widest uppercase">Description</p>
          <p className="text-xs font-semibold text-mid tracking-widest uppercase">Amount</p>
        </div>
        {txList.length === 0 && (
          <p className="px-5 py-6 text-sm text-mid">No transactions yet.</p>
        )}
        {txList.map((tx, i) => (
          <div
            key={tx.id}
            className={`flex justify-between px-5 py-3.5 text-sm ${i < txList.length - 1 ? 'border-b border-line' : ''}`}
          >
            <span className="text-ink">{tx.description}</span>
            <span className={`font-display font-bold ${tx.amountKobo >= 0 ? 'text-emerald' : 'text-red-600'}`}>
              {tx.amountKobo >= 0 ? '+' : ''}₦{naira(tx.amountKobo)}
            </span>
          </div>
        ))}
      </Card>
    </>
  );
}
