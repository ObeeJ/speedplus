export type RxStatus =
  | 'uploaded'
  | 'under_review'
  | 'approved'
  | 'rejected'
  | 'expired';

export interface Prescription {
  id: string;
  customerId: string;
  pharmacyId: string;
  status: RxStatus;
  imageUrl: string;
  uploadedAt: string;
  reviewedAt?: string;
  reviewedBy?: string;
  rejectionReason?: string;
  expiresAt?: string;
  items?: PrescriptionItem[];
}

export interface PrescriptionItem {
  productId: string;
  productName: string;
  dosage: string;
  quantity: number;
  refillsRemaining?: number;
}
