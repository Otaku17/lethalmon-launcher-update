import { useState, KeyboardEvent } from 'react';
import './Chips.css';

const CloseIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={3} strokeLinecap="square">
    <path d="M18 6 6 18M6 6l12 12" />
  </svg>
);

export interface ChipsProps {
  defaultTags?: string[];
  placeholder?: string;
  onChange?: (tags: string[]) => void;
}

/**
 * <Chips defaultTags={['NAV', 'PWR']} placeholder="+ AJOUTER" onChange={fn} />
 */
export default function Chips({ defaultTags = [], placeholder = '+ AJOUTER', onChange }: ChipsProps) {
  const [tags, setTags] = useState<string[]>(defaultTags);
  const [draft, setDraft] = useState('');

  function commit(next: string[]) {
    setTags(next);
    onChange?.(next);
  }

  function handleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' && draft.trim()) {
      e.preventDefault();
      commit([...tags, draft.trim().toUpperCase()]);
      setDraft('');
    }
  }

  function removeTag(idx: number) {
    commit(tags.filter((_, i) => i !== idx));
  }

  return (
    <div className="hud-chips">
      {tags.map((tag, idx) => (
        <span className="hud-chip" key={`${tag}-${idx}`}>
          {tag}
          <button type="button" aria-label={`Retirer ${tag}`} onClick={() => removeTag(idx)}>
            <CloseIcon />
          </button>
        </span>
      ))}
      <input
        type="text"
        className="hud-chips__input"
        placeholder={placeholder}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={handleKeyDown}
      />
    </div>
  );
}
