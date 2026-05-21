import { useEffect, useId, useMemo, useRef, useState } from "react";

type DateTimeParts = {
  date: string;
  hour: string;
  minute: string;
  second: string;
};

type DateTimeFieldProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  max?: Date;
  /** When true, the latest allowed moment is evaluated as "now" on each interaction. */
  capAtNow?: boolean;
};

const WEEKDAY_LABELS = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"];

function emptyParts(): DateTimeParts {
  return { date: "", hour: "", minute: "", second: "" };
}

function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

function digitsOnly(raw: string): string {
  return raw.replace(/\D/g, "").slice(0, 2);
}

function finalizeTimePart(raw: string, max: number): string {
  if (raw === "") {
    return "00";
  }
  const parsed = Number(raw);
  if (Number.isNaN(parsed)) {
    return "00";
  }
  return pad2(Math.min(Math.max(0, parsed), max));
}

function parseValue(value: string): DateTimeParts {
  if (!value) {
    return emptyParts();
  }

  const match = /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?/.exec(value);
  if (match) {
    return {
      date: match[1],
      hour: match[2],
      minute: match[3],
      second: match[4] ?? "00"
    };
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return emptyParts();
  }

  return {
    date: `${parsed.getFullYear()}-${pad2(parsed.getMonth() + 1)}-${pad2(parsed.getDate())}`,
    hour: pad2(parsed.getHours()),
    minute: pad2(parsed.getMinutes()),
    second: pad2(parsed.getSeconds())
  };
}

function buildValue(parts: DateTimeParts): string {
  if (!parts.date) {
    return "";
  }

  return `${parts.date}T${finalizeTimePart(parts.hour, 23)}:${finalizeTimePart(parts.minute, 59)}:${finalizeTimePart(
    parts.second,
    59
  )}`;
}

function partsToDate(parts: DateTimeParts): Date | null {
  if (!parts.date) {
    return null;
  }

  const [year, month, day] = parts.date.split("-").map(Number);
  const hour = parts.hour === "" ? 0 : Number(parts.hour);
  const minute = parts.minute === "" ? 0 : Number(parts.minute);
  const second = parts.second === "" ? 0 : Number(parts.second);

  if ([hour, minute, second].some((value) => Number.isNaN(value))) {
    return null;
  }

  return new Date(year, month - 1, day, hour, minute, second);
}

function dateToParts(date: Date): DateTimeParts {
  return {
    date: `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`,
    hour: pad2(date.getHours()),
    minute: pad2(date.getMinutes()),
    second: pad2(date.getSeconds())
  };
}

function clampPartsToMax(parts: DateTimeParts, max?: Date): DateTimeParts {
  if (!max || !parts.date) {
    return parts;
  }

  const current = partsToDate(parts);
  if (!current || current.getTime() <= max.getTime()) {
    return parts;
  }

  return dateToParts(max);
}

function isDayAfterMax(year: number, month: number, day: number, max?: Date): boolean {
  if (!max) {
    return false;
  }

  const candidate = new Date(year, month, day);
  const limit = new Date(max.getFullYear(), max.getMonth(), max.getDate());
  return candidate.getTime() > limit.getTime();
}

function canShowNextMonth(viewYear: number, viewMonth: number, max?: Date): boolean {
  if (!max) {
    return true;
  }

  if (viewYear > max.getFullYear()) {
    return false;
  }

  return viewYear < max.getFullYear() || viewMonth < max.getMonth();
}

function getTimeFieldMax(
  field: "hour" | "minute" | "second",
  parts: DateTimeParts,
  max?: Date
): number {
  if (!max || !parts.date) {
    return field === "hour" ? 23 : 59;
  }

  const [year, month, day] = parts.date.split("-").map(Number);
  const isMaxDay =
    year === max.getFullYear() && month === max.getMonth() + 1 && day === max.getDate();

  if (!isMaxDay) {
    return field === "hour" ? 23 : 59;
  }

  const hour = Number(parts.hour || 0);
  const minute = Number(parts.minute || 0);

  if (field === "hour") {
    return max.getHours();
  }

  if (hour < max.getHours()) {
    return 59;
  }

  if (field === "minute") {
    return max.getMinutes();
  }

  if (minute < max.getMinutes()) {
    return 59;
  }

  return max.getSeconds();
}

function formatDisplayDate(isoDate: string): string {
  const [year, month, day] = isoDate.split("-").map(Number);
  if (!year || !month || !day) {
    return "Select date";
  }
  return new Intl.DateTimeFormat(undefined, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric"
  }).format(new Date(year, month - 1, day));
}

function monthLabel(year: number, month: number): string {
  return new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric" }).format(
    new Date(year, month, 1)
  );
}

function mondayFirstWeekday(year: number, month: number): number {
  const weekday = new Date(year, month, 1).getDay();
  return weekday === 0 ? 6 : weekday - 1;
}

function buildCalendarDays(year: number, month: number): Array<number | null> {
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const leading = mondayFirstWeekday(year, month);
  const cells: Array<number | null> = [];

  for (let index = 0; index < leading; index += 1) {
    cells.push(null);
  }
  for (let day = 1; day <= daysInMonth; day += 1) {
    cells.push(day);
  }
  while (cells.length % 7 !== 0) {
    cells.push(null);
  }

  return cells;
}

function resolveMax(max: Date | undefined, capAtNow: boolean | undefined): Date | undefined {
  if (max) {
    return max;
  }
  if (capAtNow) {
    return new Date();
  }
  return undefined;
}

export function DateTimeField({ label, value, onChange, max, capAtNow }: DateTimeFieldProps) {
  const limit = () => resolveMax(max, capAtNow);
  const fieldId = useId();
  const popoverRef = useRef<HTMLDivElement>(null);
  const timeEditingRef = useRef(false);
  const [draft, setDraft] = useState<DateTimeParts>(() => parseValue(value));

  const initialAnchor = draft.date
    ? (() => {
        const [year, month] = draft.date.split("-").map(Number);
        return { year, month: month - 1 };
      })()
    : (() => {
        const anchor = limit() ?? new Date();
        return { year: anchor.getFullYear(), month: anchor.getMonth() };
      })();

  const [open, setOpen] = useState(false);
  const [viewYear, setViewYear] = useState(initialAnchor.year);
  const [viewMonth, setViewMonth] = useState(initialAnchor.month);

  useEffect(() => {
    if (!timeEditingRef.current) {
      setDraft(clampPartsToMax(parseValue(value), limit()));
    }
  }, [capAtNow, max, value]);

  useEffect(() => {
    if (!draft.date) {
      return;
    }
    const [year, month] = draft.date.split("-").map(Number);
    setViewYear(year);
    setViewMonth(month - 1);
  }, [draft.date]);

  useEffect(() => {
    if (!open) {
      return;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (!popoverRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  const calendarDays = useMemo(
    () => buildCalendarDays(viewYear, viewMonth),
    [viewMonth, viewYear]
  );

  const commitDraft = (next: DateTimeParts) => {
    const clamped = clampPartsToMax(next, limit());
    setDraft(clamped);
    onChange(buildValue(clamped));
  };

  const updateTimePart = (field: "hour" | "minute" | "second", raw: string) => {
    const nextValue = digitsOnly(raw);
    const next = { ...draft, [field]: nextValue };
    setDraft(next);

    if (nextValue.length === 2 && next.date) {
      const fieldMax = getTimeFieldMax(field, next, limit());
      commitDraft({ ...next, [field]: finalizeTimePart(nextValue, fieldMax) });
    }
  };

  const finalizeTimeField = (field: "hour" | "minute" | "second") => {
    const fieldMax = getTimeFieldMax(field, draft, limit());
    const finalized = finalizeTimePart(draft[field], fieldMax);
    const next = { ...draft, [field]: finalized };
    timeEditingRef.current = false;
    commitDraft(next);
  };

  const selectDay = (day: number) => {
    if (isDayAfterMax(viewYear, viewMonth, day, limit())) {
      return;
    }

    const nextDate = `${viewYear}-${pad2(viewMonth + 1)}-${pad2(day)}`;
    commitDraft({ ...draft, date: nextDate });
    setOpen(false);
  };

  const shiftMonth = (delta: number) => {
    if (delta > 0 && !canShowNextMonth(viewYear, viewMonth, limit())) {
      return;
    }

    const anchor = new Date(viewYear, viewMonth + delta, 1);
    setViewYear(anchor.getFullYear());
    setViewMonth(anchor.getMonth());
  };

  const selectedDay = draft.date
    ? (() => {
        const [year, month, day] = draft.date.split("-").map(Number);
        if (year === viewYear && month - 1 === viewMonth) {
          return day;
        }
        return null;
      })()
    : null;

  return (
    <div className="datetime-range-block">
      <span className="datetime-field-label">{label}</span>
      <div className="datetime-field-row" ref={popoverRef}>
        <div className="datetime-date-anchor">
          <button
            className="datetime-date-button"
            type="button"
            aria-expanded={open}
            aria-haspopup="dialog"
            onClick={() => setOpen((current) => !current)}
          >
            {draft.date ? formatDisplayDate(draft.date) : "Select date"}
          </button>
          {open && (
            <div
              className="datetime-calendar"
              role="dialog"
              aria-label={`${label} calendar`}
              onWheel={(event) => event.preventDefault()}
            >
              <div className="datetime-calendar-header">
                <button
                  className="datetime-calendar-nav"
                  type="button"
                  aria-label="Previous month"
                  onClick={() => shiftMonth(-1)}
                >
                  ‹
                </button>
                <span className="datetime-calendar-title">{monthLabel(viewYear, viewMonth)}</span>
                <button
                  className="datetime-calendar-nav"
                  type="button"
                  aria-label="Next month"
                  onClick={() => shiftMonth(1)}
                  disabled={!canShowNextMonth(viewYear, viewMonth, limit())}
                >
                  ›
                </button>
              </div>
              <div className="datetime-calendar-weekdays">
                {WEEKDAY_LABELS.map((weekday) => (
                  <span key={weekday}>{weekday}</span>
                ))}
              </div>
              <div className="datetime-calendar-grid">
                {calendarDays.map((day, index) =>
                  day ? (
                    <button
                      key={`${viewYear}-${viewMonth}-${day}-${index}`}
                      type="button"
                      className={
                        day === selectedDay
                          ? "datetime-calendar-day selected"
                          : isDayAfterMax(viewYear, viewMonth, day, limit())
                            ? "datetime-calendar-day disabled"
                            : "datetime-calendar-day"
                      }
                      onClick={() => selectDay(day)}
                      disabled={isDayAfterMax(viewYear, viewMonth, day, limit())}
                    >
                      {day}
                    </button>
                  ) : (
                    <span key={`empty-${index}`} className="datetime-calendar-day empty" aria-hidden />
                  )
                )}
              </div>
            </div>
          )}
        </div>
        <div className="datetime-time-inputs">
          <input
            id={`${fieldId}-hour`}
            className="datetime-time-part"
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            placeholder="HH"
            aria-label={`${label} hour`}
            value={draft.hour}
            onFocus={(event) => {
              timeEditingRef.current = true;
              event.currentTarget.select();
            }}
            onChange={(event) => updateTimePart("hour", event.target.value)}
            onBlur={() => finalizeTimeField("hour")}
          />
          <span className="datetime-time-separator" aria-hidden>
            :
          </span>
          <input
            id={`${fieldId}-minute`}
            className="datetime-time-part"
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            placeholder="MM"
            aria-label={`${label} minute`}
            value={draft.minute}
            onFocus={(event) => {
              timeEditingRef.current = true;
              event.currentTarget.select();
            }}
            onChange={(event) => updateTimePart("minute", event.target.value)}
            onBlur={() => finalizeTimeField("minute")}
          />
          <span className="datetime-time-separator" aria-hidden>
            :
          </span>
          <input
            id={`${fieldId}-second`}
            className="datetime-time-part"
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            placeholder="SS"
            aria-label={`${label} second`}
            value={draft.second}
            onFocus={(event) => {
              timeEditingRef.current = true;
              event.currentTarget.select();
            }}
            onChange={(event) => updateTimePart("second", event.target.value)}
            onBlur={() => finalizeTimeField("second")}
          />
        </div>
      </div>
    </div>
  );
}
