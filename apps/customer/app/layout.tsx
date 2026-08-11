import type { Metadata, Viewport } from 'next';
import { Instrument_Sans, Space_Grotesk } from 'next/font/google';
import { Providers } from './providers';
import { AuthGuard } from './components/auth-guard';
import './globals.css';

const instrumentSans = Instrument_Sans({ subsets: ['latin'], variable: '--font-body', display: 'swap' });
const spaceGrotesk = Space_Grotesk({
  subsets: ['latin'],
  variable: '--font-display',
  weight: ['400', '500', '600', '700'],
  display: 'swap',
});

export const viewport: Viewport = { themeColor: '#0A3D2C' };

export const metadata: Metadata = {
  title: 'SpeedPlus — Faster. Cheaper. Better.',
  description: 'Gas, groceries, food and pharmacy delivered fast across Nigeria.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${instrumentSans.variable} ${spaceGrotesk.variable}`} suppressHydrationWarning>
      <body>
        <Providers>
          <AuthGuard>{children}</AuthGuard>
        </Providers>
      </body>
    </html>
  );
}
