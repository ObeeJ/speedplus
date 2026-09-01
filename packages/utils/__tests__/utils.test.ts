import {
  formatCurrency,
  formatDate,
  formatDistance,
  formatEta,
  isValidPhone,
  isValidEmail,
  isValidPrescriptionFile,
} from '@fourdat/utils';

describe('formatCurrency', () => {
  it('formats kobo as naira', () => {
    expect(formatCurrency({ amount: 500000, currency: 'NGN' })).toContain('5,000');
  });
  it('formats zero', () => {
    expect(formatCurrency({ amount: 0, currency: 'NGN' })).toContain('0');
  });
});

describe('formatDistance', () => {
  it('shows metres under 1km', () => {
    expect(formatDistance(0.5)).toBe('500m away');
  });
  it('shows km at 1km and above', () => {
    expect(formatDistance(3.2)).toBe('3.2km away');
  });
});

describe('formatEta', () => {
  it('shows minutes under an hour', () => {
    expect(formatEta(25)).toBe('25 min');
  });
  it('shows hours and minutes', () => {
    expect(formatEta(90)).toBe('1h 30min');
  });
  it('shows whole hours', () => {
    expect(formatEta(120)).toBe('2h');
  });
});

describe('formatDate', () => {
  it('returns a non-empty string for a valid ISO date', () => {
    expect(formatDate('2026-07-31T12:00:00Z').length).toBeGreaterThan(0);
  });
});

describe('isValidPhone', () => {
  it('accepts valid Nigerian numbers', () => {
    expect(isValidPhone('08012345678')).toBe(true);
    expect(isValidPhone('+2348012345678')).toBe(true);
    expect(isValidPhone('07012345678')).toBe(true);
  });
  it('rejects invalid numbers', () => {
    expect(isValidPhone('1234')).toBe(false);
    expect(isValidPhone('0601234567')).toBe(false);
  });
  it('strips spaces before validating', () => {
    expect(isValidPhone('0801 234 5678')).toBe(true);
  });
});

describe('isValidEmail', () => {
  it('accepts valid emails', () => {
    expect(isValidEmail('user@example.com')).toBe(true);
  });
  it('rejects invalid emails', () => {
    expect(isValidEmail('notanemail')).toBe(false);
    expect(isValidEmail('@nodomain')).toBe(false);
  });
});

describe('isValidPrescriptionFile', () => {
  function makeFile(type: string, size: number): File {
    return { type, size, name: 'rx' } as unknown as File;
  }

  it('accepts valid image types under 10MB', () => {
    expect(isValidPrescriptionFile(makeFile('image/jpeg', 1024)).valid).toBe(true);
    expect(isValidPrescriptionFile(makeFile('image/png', 1024)).valid).toBe(true);
    expect(isValidPrescriptionFile(makeFile('application/pdf', 1024)).valid).toBe(true);
  });
  it('rejects unsupported types', () => {
    const result = isValidPrescriptionFile(makeFile('image/gif', 1024));
    expect(result.valid).toBe(false);
    expect(result.error).toBeTruthy();
  });
  it('rejects files over 10MB', () => {
    const result = isValidPrescriptionFile(makeFile('image/jpeg', 11 * 1024 * 1024));
    expect(result.valid).toBe(false);
    expect(result.error).toBeTruthy();
  });
});
