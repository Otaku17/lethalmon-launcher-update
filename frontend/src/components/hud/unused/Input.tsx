import { InputHTMLAttributes } from 'react';
import './Input.css';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  prefix?: string;
}

/**
 * <Input prefix=">" placeholder="ENTER COMMAND" />
 */
export default function Input({ prefix = '>', className = '', ...rest }: InputProps) {
  return (
    <div className={`hud-input ${className}`}>
      {prefix && <span className="hud-input__prefix">{prefix}</span>}
      <input type="text" spellCheck={false} {...rest} />
    </div>
  );
}
