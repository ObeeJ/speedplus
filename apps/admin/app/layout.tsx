import type { Metadata, Viewport } from 'next';
import { Instrument_Sans, DM_Sans } from 'next/font/google';
import { Providers } from './providers';
import { AdminNav } from '../components/admin-nav';
import { AdminAuthGuard } from '../components/admin-auth-guard';
import './globals.css';

const instrumentSans = Instrument_Sans({ subsets: ['latin'], variable: '--font-body', display: 'swap' });
const dmSans = DM_Sans({ subsets: ['latin'], variable: '--font-display', weight: ['400', '500', '600', '700'], display: 'swap' });

export const viewport: Viewport = { themeColor: '#0A3D2C' };

export const metadata: Metadata = {
  title: 'SpeedPlus Ops',
  description: 'Admin panel for SpeedPlus operations.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${instrumentSans.variable} ${dmSans.variable}`} suppressHydrationWarning>
      <body>
        <Providers>
          <AdminAuthGuard>
            <div className="min-h-screen flex flex-col lg:flex-row bg-sand">
              <AdminNav />
              <main className="flex-1 min-w-0 overflow-y-auto">{children}</main>
            </div>
          </AdminAuthGuard>
        </Providers>
      </body>
    </html>
  );
}
