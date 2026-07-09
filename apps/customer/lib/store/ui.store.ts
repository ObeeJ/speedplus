import { create } from 'zustand';

interface Toast { id: string; message: string; type: 'success' | 'error' | 'info' }

interface UiState {
  toasts: Toast[];
  isCartOpen: boolean;
  addToast: (message: string, type?: Toast['type']) => void;
  removeToast: (id: string) => void;
  setCartOpen: (open: boolean) => void;
}

export const useUiStore = create<UiState>()((set) => ({
  toasts: [],
  isCartOpen: false,
  addToast: (message, type = 'info') => {
    const id = crypto.randomUUID();
    set((s) => ({ toasts: [...s.toasts, { id, message, type }] }));
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), 4000);
  },
  removeToast: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
  setCartOpen: (open) => set({ isCartOpen: open }),
}));
