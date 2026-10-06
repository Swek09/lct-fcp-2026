import {
  forwardRef,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type FocusEvent,
  type KeyboardEvent,
} from "react";

export function formatPhone(input: string): string {
  if (!input) return "";
  const trimmed = input.trim();
  if (!trimmed) return "";

  const startsWithPlus = trimmed.startsWith("+");
  const digits = trimmed.replace(/\D/g, "");

  if (!digits) {
    return startsWithPlus ? "+" : "";
  }

  // Check if international non-Russian (+ and country code not 7)
  const isInternationalNonRussian = startsWithPlus && !trimmed.startsWith("+7");

  if (isInternationalNonRussian) {
    const limited = digits.slice(0, 15);
    if (limited.length <= 3) return "+" + limited;
    if (limited.length <= 6) return "+" + limited.slice(0, 3) + " " + limited.slice(3);
    if (limited.length <= 10) {
      return "+" + limited.slice(0, 3) + " " + limited.slice(3, 6) + " " + limited.slice(6);
    }
    return (
      "+" +
      limited.slice(0, 3) +
      " " +
      limited.slice(3, 6) +
      " " +
      limited.slice(6, 10) +
      " " +
      limited.slice(10)
    );
  }

  // Russian / Kazakh format (+7)
  let nationalDigits = digits;
  if (nationalDigits.startsWith("7") || nationalDigits.startsWith("8")) {
    nationalDigits = nationalDigits.slice(1);
  }
  nationalDigits = nationalDigits.slice(0, 10);

  let res = "+7";
  if (nationalDigits.length > 0) {
    res += " (" + nationalDigits.slice(0, 3);
  }
  if (nationalDigits.length >= 3) {
    res += ") ";
  }
  if (nationalDigits.length > 3) {
    res += nationalDigits.slice(3, 6);
  }
  if (nationalDigits.length > 6) {
    res += "-" + nationalDigits.slice(6, 8);
  }
  if (nationalDigits.length > 8) {
    res += "-" + nationalDigits.slice(8, 10);
  }
  return res;
}

export function validatePhone(
  value: string,
  required = false,
): { valid: boolean; error?: string } {
  const trimmed = value.trim();
  if (!trimmed) {
    if (required) {
      return { valid: false, error: "Укажите номер телефона" };
    }
    return { valid: true };
  }

  if (!/^\+?[0-9\s()\-]+$/.test(trimmed)) {
    return {
      valid: false,
      error: "Номер может содержать только цифры и символы +, -, (, )",
    };
  }

  const digits = trimmed.replace(/\D/g, "");
  if (digits.length < 7) {
    return { valid: false, error: "Номер должен содержать не менее 7 цифр" };
  }
  if (digits.length > 15) {
    return { valid: false, error: "Номер должен содержать не более 15 цифр" };
  }

  const isRussian =
    trimmed.startsWith("+7") || trimmed.startsWith("8") || trimmed.startsWith("7");
  if (isRussian && (trimmed.startsWith("+7") || !trimmed.startsWith("+"))) {
    if (digits.length !== 11) {
      return {
        valid: false,
        error: `Номер должен содержать 11 цифр: +7 (XXX) XXX-XX-XX (сейчас ${digits.length})`,
      };
    }
  }

  return { valid: true };
}

function findCursorAfterDigits(formatted: string, targetDigits: number): number {
  if (targetDigits <= 0) return 0;
  let count = 0;
  let pos = formatted.length;
  for (let i = 0; i < formatted.length; i++) {
    if (/\d/.test(formatted[i])) {
      count++;
      if (count === targetDigits) {
        pos = i + 1;
        break;
      }
    }
  }
  while (pos < formatted.length && !/\d/.test(formatted[pos])) {
    pos++;
  }
  return pos;
}

export interface PhoneInputProps {
  id?: string;
  name?: string;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
  onBlur?: (e: FocusEvent<HTMLInputElement>) => void;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  className?: string;
  autoFocus?: boolean;
  maxLength?: number;
}

export const PhoneInput = forwardRef<HTMLInputElement, PhoneInputProps>(
  (
    {
      id,
      name,
      value,
      defaultValue,
      onChange,
      onBlur,
      placeholder = "+7 (900) 000-00-00",
      required = false,
      disabled = false,
      className,
      autoFocus,
      maxLength = 40,
    },
    ref,
  ) => {
    const innerRef = useRef<HTMLInputElement | null>(null);
    const generatedId = useId();
    const inputId = id || generatedId;
    const errorId = `${inputId}-error`;

    const isControlled = value !== undefined;
    const [internalVal, setInternalVal] = useState(() =>
      formatPhone(defaultValue || ""),
    );
    const [touched, setTouched] = useState(false);

    const currentDisplayVal = isControlled ? formatPhone(value || "") : internalVal;
    const validation = validatePhone(currentDisplayVal, required);
    const showError = touched && !validation.valid;

    const setSelection = (pos: number) => {
      requestAnimationFrame(() => {
        const el = (ref && "current" in ref ? ref.current : null) || innerRef.current;
        if (el) {
          el.setSelectionRange(pos, pos);
        }
      });
    };

    const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Backspace") {
        const input = (ref && "current" in ref ? ref.current : null) || innerRef.current;
        if (!input) return;
        const { selectionStart, selectionEnd, value: currentVal } = input;
        if (selectionStart !== null && selectionStart === selectionEnd) {
          const charBefore = currentVal[selectionStart - 1];
          if (charBefore && !/\d/.test(charBefore)) {
            e.preventDefault();
            let prevDigitIdx = -1;
            for (let i = selectionStart - 1; i >= 0; i--) {
              if (/\d/.test(currentVal[i])) {
                prevDigitIdx = i;
                break;
              }
            }
            if (prevDigitIdx === -1 || prevDigitIdx <= 1) {
              applyChange("", 0);
            } else {
              const digitsBefore = currentVal
                .slice(0, prevDigitIdx)
                .replace(/\D/g, "").length;
              const nextRaw =
                currentVal.slice(0, prevDigitIdx) + currentVal.slice(selectionStart);
              applyChange(nextRaw, digitsBefore);
            }
          } else if (selectionStart <= 2 && currentVal.startsWith("+7")) {
            e.preventDefault();
            applyChange("", 0);
          }
        }
      }
    };

    const applyChange = (raw: string, targetDigitsCount?: number) => {
      const formatted = formatPhone(raw);
      if (!isControlled) {
        setInternalVal(formatted);
      }
      onChange?.(formatted);

      if (targetDigitsCount !== undefined) {
        const nextPos = findCursorAfterDigits(formatted, targetDigitsCount);
        setSelection(nextPos);
      }
    };

    const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
      const raw = e.target.value;
      const selStart = e.target.selectionStart || raw.length;
      const digitsBeforeCursor = raw.slice(0, selStart).replace(/\D/g, "").length;

      applyChange(raw, digitsBeforeCursor);
      if (!touched && raw.replace(/\D/g, "").length >= 4) {
        setTouched(true);
      }
    };

    const handleBlur = (e: FocusEvent<HTMLInputElement>) => {
      setTouched(true);
      onBlur?.(e);
    };

    return (
      <div className="phone-input-group">
        <input
          ref={(node) => {
            innerRef.current = node;
            if (typeof ref === "function") {
              ref(node);
            } else if (ref) {
              ref.current = node;
            }
          }}
          id={inputId}
          name={name}
          type="tel"
          inputMode="tel"
          autoComplete="tel"
          value={currentDisplayVal}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onBlur={handleBlur}
          placeholder={placeholder}
          disabled={disabled}
          required={required}
          autoFocus={autoFocus}
          maxLength={maxLength}
          aria-invalid={showError ? "true" : "false"}
          aria-describedby={showError ? errorId : undefined}
          className={`phone-input ${showError ? "has-error" : ""} ${className || ""}`.trim()}
        />
        {showError && (
          <span className="field-error" role="alert" id={errorId}>
            {validation.error}
          </span>
        )}
      </div>
    );
  },
);

PhoneInput.displayName = "PhoneInput";
