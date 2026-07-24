import type { Metadata } from 'next';
import { Instrument_Sans, Space_Grotesk } from 'next/font/google';
import { Providers } from './providers';
import { AdminNav } from '../components/admin-nav';
import { AdminAuthGuard } from '../components/admin-auth-guard';
import './globals.css';

const instrumentSans = Instrument_Sans({ subsets: ['latin'], variable: '--font-body', display: 'swap' });
const spaceGrotesk = Space_Grotesk({ subsets: ['latin'], variable: '--font-display', weight: ['400', '500', '600', '700'], display: 'swap' });

export const metadata: Metadata = {
  title: 'SpeedPlus Ops',
  description: 'Admin panel for SpeedPlus operations.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${instrumentSans.variable} ${spaceGrotesk.variable}`}>
      <body>
        <Providers>
          <AdminAuthGuard>
            <div className="min-h-screen flex bg-sand">
              <AdminNav />
              <main className="flex-1 min-w-0 overflow-y-auto">{children}</main>
            </div>
          </AdminAuthGuard>
        </Providers>
      </body>
    </html>
  );
}
