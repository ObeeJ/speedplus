'use client';

export function SelectTile({
  label,
  description,
  selected,
  onClick,
}: {
  label: string;
  description?: string;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`w-full flex flex-col items-start gap-1 rounded-[13px] border-2 px-4 py-3.5 text-left transition-all ${
        selected ? 'bg-emerald border-lime' : 'bg-white border-line hover:border-emerald/40'
      }`}
    >
      <span className={`font-display font-semibold text-[15px] ${selected ? 'text-lime' : 'text-ink'}`}>
        {selected ? `✓ ${label}` : label}
      </span>
      {description && <span className={`text-[12.5px] ${selected ? 'text-sand/70' : 'text-mid'}`}>{description}</span>}
    </button>
  );
}
