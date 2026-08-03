import { useState, ReactNode } from 'react';
import './Accordion.css';

const ChevronIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="square">
    <path d="m6 9 6 6 6-6" />
  </svg>
);

export interface AccordionItem {
  id: string;
  title: string;
  content: ReactNode;
}

export interface AccordionProps {
  items: AccordionItem[];
  defaultOpenId?: string | null;
}

/**
 * <Accordion
 *   items={[{ id:'log', title:'Journal de vol', content:'...' }, ...]}
 *   defaultOpenId="log"
 * />
 */
export default function Accordion({ items, defaultOpenId = null }: AccordionProps) {
  const [openId, setOpenId] = useState<string | null>(defaultOpenId);

  return (
    <div className="hud-accordion">
      {items.map((item) => {
        const isOpen = item.id === openId;
        return (
          <div className={`hud-accordion__item ${isOpen ? 'is-open' : ''}`} key={item.id}>
            <button
              className="hud-accordion__head"
              type="button"
              onClick={() => setOpenId(isOpen ? null : item.id)}
            >
              <span>{item.title}</span>
              <ChevronIcon />
            </button>
            <div
              className="hud-accordion__panel"
              style={{ maxHeight: isOpen ? '240px' : '0px' }}
            >
              <div className="hud-accordion__panel-inner">{item.content}</div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
