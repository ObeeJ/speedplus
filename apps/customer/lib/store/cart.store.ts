import { create } from 'zustand';
import type { Vertical } from '@fourdat/types';

interface CartItem {
  productId: string;
  name: string;
  price: number;
  quantity: number;
  customizations?: string;
  imageUrl?: string;
}

interface CartState {
  merchantId: string | null;
  vertical: Vertical | null;
  items: CartItem[];
  addItem: (merchantId: string, vertical: Vertical, item: CartItem) => void;
  removeItem: (productId: string) => void;
  updateQuantity: (productId: string, quantity: number) => void;
  clearCart: () => void;
  totalItems: () => number;
  subtotal: () => number;
}

export const useCartStore = create<CartState>()((set, get) => ({
  merchantId: null,
  vertical: null,
  items: [],

  addItem: (merchantId, vertical, item) => {
    const s = get();
    if (s.merchantId && s.merchantId !== merchantId) {
      set({ merchantId, vertical, items: [item] });
      return;
    }
    const existing = s.items.find((i) => i.productId === item.productId);
    set({
      merchantId,
      vertical,
      items: existing
        ? s.items.map((i) => i.productId === item.productId ? { ...i, quantity: i.quantity + item.quantity } : i)
        : [...s.items, item],
    });
  },

  removeItem: (productId) => set((s) => ({ items: s.items.filter((i) => i.productId !== productId) })),

  updateQuantity: (productId, quantity) =>
    set((s) => ({
      items: quantity <= 0
        ? s.items.filter((i) => i.productId !== productId)
        : s.items.map((i) => i.productId === productId ? { ...i, quantity } : i),
    })),

  clearCart: () => set({ merchantId: null, vertical: null, items: [] }),
  totalItems: () => get().items.reduce((sum, i) => sum + i.quantity, 0),
  subtotal: () => get().items.reduce((sum, i) => sum + i.price * i.quantity, 0),
}));
