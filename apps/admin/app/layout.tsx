import type { Metadata } from 'next';
import { Inter, Plus_Jakarta_Sans } from 'next/font/google';
import { Providers } from './providers';
import './globals.css';

const inter = Inter({ subsets: ['latin'], variable: '--font-body', display: 'swap' });
const plusJakarta = Plus_Jakarta_Sans({ subsets: ['latin'], variable: '--font-display', weight: ['400', '500', '600', '700', '800'], display: 'swap' });

export const metadata: Metadata = {
  title: 'SpeedPlus Admin',
  description: 'Admin panel for SpeedPlus operations.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${inter.variable} ${plusJakarta.variable}`}>
      <body><Providers>{children}</Providers></body>
    </html>
  );
}
