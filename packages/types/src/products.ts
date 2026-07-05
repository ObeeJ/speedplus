import type { Money, Vertical } from './common';

export interface BaseProduct {
  id: string;
  merchantId: string;
  vertical: Vertical;
  name: string;
  description?: string;
  price: Money;
  imageUrl?: string;
  isAvailable: boolean;
  category: string;
}

export type CylinderSize = '3kg' | '5kg' | '6kg' | '12.5kg' | '25kg' | '50kg';

export interface GasProduct extends BaseProduct {
  vertical: 'gas';
  cylinderSize: CylinderSize;
  isRefill: boolean;
  requiresEmptyReturn: boolean;
}

export interface GroceryProduct extends BaseProduct {
  vertical: 'grocery';
  unit?: string;
  isWeighted: boolean;
  isAgeRestricted: boolean;
  allergens?: string[];
}

export interface FoodItem extends BaseProduct {
  vertical: 'food';
  modifierGroups?: ModifierGroup[];
  prepTimeMinutes: number;
  isVegetarian: boolean;
  isHalal: boolean;
  calories?: number;
}

export interface ModifierGroup {
  id: string;
  name: string;
  required: boolean;
  minSelections: number;
  maxSelections: number;
  options: ModifierOption[];
}

export interface ModifierOption {
  id: string;
  name: string;
  price: Money;
}

export type RxRequirement = 'none' | 'otc' | 'prescription';

export interface PharmacyProduct extends BaseProduct {
  vertical: 'pharmacy';
  rxRequirement: RxRequirement;
  genericName?: string;
  dosage?: string;
  requiresColdChain: boolean;
}

export type Product = GasProduct | GroceryProduct | FoodItem | PharmacyProduct;
