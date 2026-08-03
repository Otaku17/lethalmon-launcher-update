import './PathInput.css';

const FolderIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="square">
    <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z" />
  </svg>
);

export interface PathInputProps {
  /** Current path to display (read-only). */
  value: string;
  /** Text shown when `value` is empty. */
  placeholder?: string;
  /** Called on icon click — it's up to you to open the folder picker
   *  (Electron `dialog.showOpenDialog`, Wails `runtime.OpenDirectoryDialog`,
   *  File System Access API `showDirectoryPicker()`, etc.) and update `value`. */
  onBrowse?: () => void;
  disabled?: boolean;
  ariaLabel?: string;
}

/**
 * <PathInput
 *   value={folderPath}
 *   placeholder="Aucun dossier sélectionné"
 *   onBrowse={handleBrowse}
 * />
 */
export default function PathInput({
  value,
  placeholder = 'Aucun dossier sélectionné',
  onBrowse,
  disabled = false,
  ariaLabel = 'Parcourir pour sélectionner un dossier',
}: PathInputProps) {
  return (
    <div className={`hud-path ${disabled ? 'is-disabled' : ''}`}>
      <span className="hud-path__value" title={value || undefined}>
        {value
          ? <span className="hud-path__text">{value}</span>
          : <span className="hud-path__placeholder">{placeholder}</span>}
      </span>
      {onBrowse && (
        <button
          type="button"
          className="hud-path__browse"
          onClick={onBrowse}
          disabled={disabled}
          aria-label={ariaLabel}
        >
          <FolderIcon />
        </button>
      )}
    </div>
  );
}
