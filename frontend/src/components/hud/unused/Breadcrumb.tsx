import { Fragment } from 'react';
import './Breadcrumb.css';

export interface BreadcrumbItem {
  label: string;
  href?: string;
}

export interface BreadcrumbProps {
  items: BreadcrumbItem[];
}

/**
 * <Breadcrumb items={[{ label:'SYSTÈME', href:'#' }, { label:'NAVIGATION', href:'#' }, { label:'CARTOGRAPHIE' }]} />
 * (le dernier élément sans `href` est traité comme la page courante)
 */
export default function Breadcrumb({ items }: BreadcrumbProps) {
  return (
    <nav className="hud-breadcrumb" aria-label="Fil d'ariane">
      {items.map((item, idx) => (
        <Fragment key={idx}>
          {item.href ? <a href={item.href}>{item.label}</a> : <span className="is-current">{item.label}</span>}
          {idx < items.length - 1 && <span className="hud-breadcrumb__sep">/</span>}
        </Fragment>
      ))}
    </nav>
  );
}
