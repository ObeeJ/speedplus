const NIGERIA_PHONE_REGEX = /^(\+234|0)[789][01]\d{8}$/;
const ALLOWED_RX_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'application/pdf'];
const MAX_RX_SIZE_BYTES = 10 * 1024 * 1024;

export function isValidPhone(phone: string): boolean {
  return NIGERIA_PHONE_REGEX.test(phone.replace(/\s/g, ''));
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

export function isValidPrescriptionFile(file: File): { valid: boolean; error?: string } {
  if (!ALLOWED_RX_TYPES.includes(file.type)) {
    return { valid: false, error: 'File must be JPG, PNG, WebP, or PDF' };
  }
  if (file.size > MAX_RX_SIZE_BYTES) {
    return { valid: false, error: 'File must be smaller than 10MB' };
  }
  return { valid: true };
}
