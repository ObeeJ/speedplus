import { SpeedPlusLogo } from '@speedplus/ui';

const verticals = [
  {
    emoji: '🔥',
    label: 'Cooking Gas',
    description: 'Cylinder refills & swaps delivered to your door. Never run out.',
    accent: '#00C48C',
    href: '/gas',
  },
  {
    emoji: '🛒',
    label: 'Grocery',
    description: 'Fresh produce, pantry staples, and household essentials.',
    accent: '#00C48C',
    href: '/grocery',
  },
  {
    emoji: '🍽️',
    label: 'Food',
    description: 'Hot meals from local restaurants, delivered fast.',
    accent: '#00C48C',
    href: '/food',
  },
  {
    emoji: '💊',
    label: 'Pharmacy',
    description: 'OTC meds and prescription fulfillment. Upload your Rx.',
    accent: '#00C48C',
    href: '/pharmacy',
  },
];

export default function HomePage() {
  return (
    <main className="min-h-screen bg-midnight text-white">
      <div className="mx-auto max-w-5xl px-6 py-16 flex flex-col items-center gap-12">

        <header className="flex flex-col items-center gap-4 text-center">
          <SpeedPlusLogo variant="full" theme="dark" size="xl" />
          <p className="text-mid text-lg max-w-md">
            Faster. Cheaper. Better. Essential delivery for everyday Nigeria.
          </p>
        </header>

        <section className="w-full">
          <h2 className="text-center text-sm font-semibold uppercase tracking-widest text-mid mb-6">
            What do you need?
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {verticals.map((v) => (
              <a
                key={v.href}
                href={v.href}
                className="group rounded-2xl border border-white/10 bg-white/5 p-6 hover:bg-white/10 hover:border-primary/40 transition-all duration-200"
              >
                <div className="text-4xl mb-3">{v.emoji}</div>
                <h3 className="font-display font-bold text-xl text-white mb-1 group-hover:text-primary transition-colors">
                  {v.label}
                </h3>
                <p className="text-mid text-sm leading-relaxed">{v.description}</p>
              </a>
            ))}
          </div>
        </section>

        <footer className="text-center text-xs text-mid/60 mt-8">
          Built for the rest of us. &copy; {new Date().getFullYear()} SpeedPlus
        </footer>
      </div>
    </main>
  );
}
