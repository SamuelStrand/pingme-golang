type ToggleProps = {
  id?: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  description?: string;
  disabled?: boolean;
  className?: string;
};

export function Toggle({
  id,
  checked,
  onChange,
  label,
  description,
  disabled,
  className
}: ToggleProps) {
  return (
    <label className={`toggle-field ${className || ""}`.trim()} htmlFor={id}>
      <span className="toggle-copy">
        <span className="toggle-label">{label}</span>
        {description && <span className="toggle-description">{description}</span>}
      </span>
      <button
        id={id}
        type="button"
        role="switch"
        aria-checked={checked}
        className={`toggle-switch ${checked ? "on" : ""}`}
        disabled={disabled}
        onClick={() => onChange(!checked)}
      >
        <span className="toggle-thumb" />
      </button>
    </label>
  );
}
