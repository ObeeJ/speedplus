import { cn } from '../src/lib/utils';

describe('cn', () => {
  it('joins class names', () => {
    expect(cn('a', 'b')).toBe('a b');
  });

  it('ignores falsy values', () => {
    expect(cn('a', false && 'b', undefined, null, 'c')).toBe('a c');
  });

  it('merges conflicting tailwind classes (last wins)', () => {
    // tailwind-merge: p-2 overrides p-4
    expect(cn('p-4', 'p-2')).toBe('p-2');
  });

  it('handles conditional object syntax', () => {
    expect(cn({ 'text-red-500': true, 'text-green-500': false })).toBe('text-red-500');
  });

  it('returns empty string for no args', () => {
    expect(cn()).toBe('');
  });
});
