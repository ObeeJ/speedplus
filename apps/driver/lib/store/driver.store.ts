import { create } from 'zustand';

export type DriverTab = 'home' | 'job' | 'earn' | 'me';

export interface DriverOffer {
  offerId: string;
  orderId: string;
  vertical: string;
  totalKobo: number;
  pickupAddress: string;
  dropoffAddress: string;
  distanceKm: number;
  weightKg?: number;
  sizeCategory?: string;
  stopCount?: number; // >1 = multi-drop
}

export interface JobStop {
  sequence: number;
  addressId: string;
  recipientName?: string;
  recipientPhone?: string;
  notes?: string;
  status: 'pending' | 'confirmed';
}

export interface ActiveJob {
  orderId: string;
  vertical: string; // gas|food|grocery|pharmacy|package
  // stage: 1=ride to pickup, 2=arrived pickup, 3=picked up,
  //        4=at stop (multi) or at dropoff (single), 5=pod, 6=done
  stage: number;
  customerName: string;
  customerPhone: string;
  pickupAddress: string;
  dropoffAddress: string;
  totalKobo: number;
  deliveryCode: string;
  paymentMethod: string;
  stops: JobStop[]; // empty = single drop-off
  currentStopIndex: number; // index into stops array for multi-drop
}

interface DriverState {
  tab: DriverTab;
  online: boolean;
  pendingOffer: DriverOffer | null;
  activeJob: ActiveJob | null;
  setTab: (t: DriverTab) => void;
  setOnline: (v: boolean) => void;
  setPendingOffer: (o: DriverOffer | null) => void;
  setActiveJob: (j: ActiveJob | null) => void;
  advanceJobStage: () => void;
  confirmStop: (sequence: number) => void;
  clearJob: () => void;
}

export const useDriverStore = create<DriverState>()((set, get) => ({
  tab: 'home',
  online: false,
  pendingOffer: null,
  activeJob: null,

  setTab: (t) => set({ tab: t }),
  setOnline: (v) => set({ online: v }),
  setPendingOffer: (o) => set({ pendingOffer: o }),
  setActiveJob: (j) => set({ activeJob: j }),

  advanceJobStage: () => {
    const job = get().activeJob;
    if (!job) return;
    if (job.stage >= 6) {
      set({ activeJob: null, tab: 'home' });
    } else {
      // For multi-drop: after confirming a stop, advance to next stop or done
      const nextStopIndex = job.currentStopIndex + 1;
      if (job.stops.length > 0 && job.stage === 5 && nextStopIndex < job.stops.length) {
        set({ activeJob: { ...job, stage: 4, currentStopIndex: nextStopIndex } });
      } else {
        set({ activeJob: { ...job, stage: job.stage + 1 } });
      }
    }
  },

  confirmStop: (sequence) => {
    const job = get().activeJob;
    if (!job) return;
    set({
      activeJob: {
        ...job,
        stops: job.stops.map((s) =>
          s.sequence === sequence ? { ...s, status: 'confirmed' as const } : s,
        ),
      },
    });
  },

  clearJob: () => set({ activeJob: null, pendingOffer: null }),
}));
