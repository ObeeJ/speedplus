import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface Meal {
  id: string;
  name: string;
  kitchen: string;
  prepTime: string;
  price: number;
}

export const MEALS: Meal[] = [
  { id: 'jollof', name: 'Jollof rice & chicken', kitchen: 'Kilimanjaro', prepTime: '15–20 min', price: 3500 },
  { id: 'egusi', name: 'Egusi soup & pounded yam', kitchen: "Mama Nkechi's Kitchen", prepTime: '20–25 min', price: 4200 },
  { id: 'suya', name: 'Beef suya platter', kitchen: 'Suya Spot Lekki', prepTime: '10–15 min', price: 2800 },
  { id: 'friedrice', name: 'Fried rice & turkey', kitchen: 'Kilimanjaro', prepTime: '15–20 min', price: 3800 },
];

const BASE_FARE = 100;
const PER_KM = 45;
const DEMO_KM = 3.2;

interface FoodFlowState {
  mealId: string | null;
  deliverTo: string | null;
  orderId: string | null;
  setMealId: (v: string) => void;
  setDeliverTo: (v: string) => void;
  setOrderId: (v: string | null) => void;
  reset: () => void;
  km: () => number;
  meal: () => Meal | null;
  priceBreakdown: () => { base: number; distance: number; item: number; total: number };
}

export const useFoodFlowStore = create<FoodFlowState>()(
  persist(
    (set, get) => ({
      mealId: null,
      deliverTo: null,
      orderId: null,
      setMealId: (v) => set({ mealId: v }),
      setDeliverTo: (v) => set({ deliverTo: v }),
      setOrderId: (v) => set({ orderId: v }),
      reset: () => set({ mealId: null, deliverTo: null, orderId: null }),
      km: () => DEMO_KM,
      meal: () => MEALS.find((m) => m.id === get().mealId) ?? null,
      priceBreakdown: () => {
        const distance = Math.round(get().km() * PER_KM);
        const item = get().meal()?.price ?? 0;
        return { base: BASE_FARE, distance, item, total: BASE_FARE + distance + item };
      },
    }),
    { name: 'speedplus-food-flow' },
  ),
);
