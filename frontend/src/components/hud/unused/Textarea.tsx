import { useState, ChangeEvent, TextareaHTMLAttributes } from 'react';
import './Textarea.css';

export interface TextareaProps extends Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'onChange'> {
  onChange?: (value: string) => void;
}

/**
 * <Textarea placeholder="RÉDIGER UN RAPPORT..." maxLength={240} onChange={fn} />
 */
export default function Textarea({ maxLength = 240, onChange, ...rest }: TextareaProps) {
  const [length, setLength] = useState(0);

  function handleChange(e: ChangeEvent<HTMLTextAreaElement>) {
    setLength(e.target.value.length);
    onChange?.(e.target.value);
  }

  return (
    <div className="hud-textarea">
      <textarea maxLength={maxLength} onChange={handleChange} {...rest} />
      <span className="hud-textarea__count">{length}/{maxLength}</span>
    </div>
  );
}
