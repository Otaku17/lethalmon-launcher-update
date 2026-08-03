import { useState, ReactNode, KeyboardEvent } from 'react';
import './Tabs.css';

export interface TabItem {
  id: string;
  label: string;
  content: ReactNode;
}

export interface TabsProps {
  tabs: TabItem[];
  defaultTabId?: string;
  /** Controlled mode: active tab imposed by the parent (with `onChange`). */
  activeId?: string;
  onChange?: (id: string) => void;
  className?: string;
}

/**
 * <Tabs
 *   tabs={[
 *     { id:'nav', label:'NAV', content: <>...</> },
 *     { id:'pwr', label:'PWR', content: <>...</> },
 *   ]}
 *   defaultTabId="nav"
 * />
 *
 * Controlled mode (e.g. switching tabs triggers an external fetch):
 * <Tabs tabs={tabs} activeId={tab} onChange={setTab} />
 */
export default function Tabs({ tabs, defaultTabId, activeId, onChange, className = '' }: TabsProps) {
  const [internalId, setInternalId] = useState(defaultTabId ?? tabs[0]?.id);
  const currentId = activeId ?? internalId;
  const active = tabs.find((t) => t.id === currentId) ?? tabs[0];

  function selectTab(id: string) {
    onChange?.(id);
    if (activeId === undefined) setInternalId(id);
  }

  function handleKeyDown(e: KeyboardEvent<HTMLLIElement>, idx: number) {
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      selectTab(tabs[(idx + 1) % tabs.length].id);
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      selectTab(tabs[(idx - 1 + tabs.length) % tabs.length].id);
    }
  }

  return (
    <div className={`hud-tabs ${className}`.trim()}>
      <ul className="hud-tabs__list" role="tablist">
        {tabs.map((tab, idx) => (
          <li
            key={tab.id}
            className="hud-tabs__tab"
            role="tab"
            aria-selected={tab.id === currentId}
            tabIndex={tab.id === currentId ? 0 : -1}
            onClick={() => selectTab(tab.id)}
            onKeyDown={(e) => handleKeyDown(e, idx)}
          >
            {tab.label}
          </li>
        ))}
      </ul>
      <div className="hud-tabs__panel" role="tabpanel">
        {active?.content}
      </div>
    </div>
  );
}
