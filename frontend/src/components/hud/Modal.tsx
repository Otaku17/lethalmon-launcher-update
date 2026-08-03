import { useEffect, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import Button from './Button';
import CaptionBar from './CaptionBar';
import './Modal.css';

const CloseIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="square">
    <path d="M18 6 6 18M6 6l12 12" />
  </svg>
);

interface ModalBaseProps {
  open: boolean;
  eyebrow?: string;
  title: string;
  children: ReactNode;
  /** `medium`/`wide` for modals with more content (forms, release notes...). */
  size?: 'default' | 'medium' | 'wide';
}

export interface ModalActionsProps extends ModalBaseProps {
  /** Footer with two buttons (Cancel / Confirm). This is the default variant. */
  variant?: 'actions';
  confirmLabel?: string;
  cancelLabel?: string;
  confirmVariant?: 'default' | 'danger';
  onConfirm: () => void;
  onCancel: () => void;
}

export interface ModalCloseProps extends ModalBaseProps {
  /** No footer, just a close cross in the top-right corner. */
  variant: 'close';
  onClose: () => void;
}

export type ModalProps = ModalActionsProps | ModalCloseProps;

/**
 * Variant with buttons (default):
 * <Modal
 *   open={isOpen}
 *   eyebrow="// CONFIRMATION_REQUIRED"
 *   title="PURGE CACHE?"
 *   confirmLabel="CONFIRM"
 *   cancelLabel="CANCEL"
 *   confirmVariant="danger"
 *   onConfirm={() => setOpen(false)}
 *   onCancel={() => setOpen(false)}
 * >
 *   This clears temporary browsing data. Irreversible.
 * </Modal>
 *
 * Variant with a plain close cross:
 * <Modal
 *   variant="close"
 *   open={isOpen}
 *   title="MISSION LOG"
 *   onClose={() => setOpen(false)}
 * >
 *   Free-form content, no confirmation buttons.
 * </Modal>
 */
export default function Modal(props: ModalProps) {
  const { open, eyebrow, title, children, size = 'default' } = props;
  const dismiss = props.variant === 'close' ? props.onClose : props.onCancel;

  useEffect(() => {
    if (!open) return;
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') dismiss();
    }
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [open, dismiss]);

  return createPortal(
    <div
      className={`hud-modal-backdrop ${open ? 'is-open' : ''}`}
      onClick={(e) => { if (e.target === e.currentTarget) dismiss(); }}
    >
      <div
        className={`hud-modal ${size !== 'default' ? `hud-modal--${size}` : ''}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="hud-modal-title"
      >
        {props.variant === 'close' && (
          <button className="hud-modal__close" type="button" aria-label="Fermer" onClick={props.onClose}>
            <CloseIcon />
          </button>
        )}

        <div className={`hud-modal__body ${props.variant === 'close' ? 'hud-modal__body--with-close' : ''}`}>
          {eyebrow && <CaptionBar label={eyebrow} className="hud-modal__eyebrow" />}
          <h2 className="hud-modal__title" id="hud-modal-title">{title}</h2>
          <div className="hud-modal__text">{children}</div>
        </div>

        {props.variant !== 'close' && (
          <div className="hud-modal__actions">
            <Button variant="ghost" onClick={props.onCancel}>{props.cancelLabel ?? 'ANNULER'}</Button>
            <Button variant={props.confirmVariant ?? 'default'} onClick={props.onConfirm}>
              {props.confirmLabel ?? 'CONFIRMER'}
            </Button>
          </div>
        )}
      </div>
    </div>,
    document.body,
  );
}
