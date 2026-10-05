import * as SelectPrimitive from "@radix-ui/react-select";
import { Check, ChevronDown, ChevronUp } from "lucide-react";
import { useId } from "react";

const EMPTY_VALUE = "__fsp_all_options__";

type SelectProps = {
  label: string;
  options: Record<string, string>;
  emptyLabel?: string;
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  name?: string;
  id?: string;
  disabled?: boolean;
  required?: boolean;
};

export function Select({
  label,
  options,
  emptyLabel,
  value,
  defaultValue,
  onValueChange,
  name,
  id,
  disabled,
  required,
}: SelectProps) {
  const generatedId = useId();
  const items = Object.entries(options);
  // Radix reserves an empty string for its placeholder. Filters still expose
  // an empty value to the caller so clearing them removes the URL parameter.
  const toInternal = (value: string) =>
    value === "" && emptyLabel ? EMPTY_VALUE : value;

  return (
    <SelectPrimitive.Root
      name={name}
      value={value === undefined ? undefined : toInternal(value)}
      defaultValue={toInternal(
        defaultValue ?? (emptyLabel ? "" : (items[0]?.[0] ?? "")),
      )}
      onValueChange={(next) =>
        onValueChange?.(next === EMPTY_VALUE ? "" : next)
      }
      disabled={disabled}
      required={required}
    >
      <SelectPrimitive.Trigger
        id={id ?? generatedId}
        className="select-trigger"
        aria-label={label}
      >
        <span className="select-value">
          <SelectPrimitive.Value />
        </span>
        <SelectPrimitive.Icon asChild>
          <ChevronDown
            className="select-chevron"
            size={17}
            aria-hidden="true"
          />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>
      <SelectPrimitive.Portal>
        <SelectPrimitive.Content
          className="select-content"
          position="popper"
          sideOffset={7}
          align="start"
          collisionPadding={12}
        >
          <SelectPrimitive.ScrollUpButton className="select-scroll-button">
            <ChevronUp size={16} aria-hidden="true" />
          </SelectPrimitive.ScrollUpButton>
          <SelectPrimitive.Viewport className="select-viewport">
            {[...(emptyLabel ? [[EMPTY_VALUE, emptyLabel]] : []), ...items].map(
              ([key, text]) => (
                <SelectPrimitive.Item
                  className="select-item"
                  key={key}
                  value={key}
                >
                  <SelectPrimitive.ItemText>{text}</SelectPrimitive.ItemText>
                  <SelectPrimitive.ItemIndicator className="select-item-check">
                    <Check size={16} aria-hidden="true" />
                  </SelectPrimitive.ItemIndicator>
                </SelectPrimitive.Item>
              ),
            )}
          </SelectPrimitive.Viewport>
          <SelectPrimitive.ScrollDownButton className="select-scroll-button">
            <ChevronDown size={16} aria-hidden="true" />
          </SelectPrimitive.ScrollDownButton>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}
