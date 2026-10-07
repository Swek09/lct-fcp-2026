import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

export const badgeVariants = cva("status", {
  variants: {
    variant: {
      default: "",
      success: "published",
      warning: "sent",
      neutral: "draft",
      danger: "rejected",
      outline: "tag",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <span className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

const statusVariantsMap: Record<string, "success" | "warning" | "neutral" | "danger"> = {
  // Common / applications / invitations
  published: "success",
  shortlisted: "success",
  accepted: "success",
  new: "warning",
  sent: "warning",
  viewed: "warning",
  draft: "neutral",
  withdrawn: "neutral",
  closed: "neutral",
  rejected: "danger",
  declined: "danger",
};

export function StatusBadge({
  status,
  label,
  className,
}: {
  status: string;
  label?: string;
  className?: string;
}) {
  const variant = statusVariantsMap[status] || "neutral";
  return (
    <span className={cn("status", status, badgeVariants({ variant }), className)}>
      {label || status}
    </span>
  );
}
