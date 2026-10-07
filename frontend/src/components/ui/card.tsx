import * as React from "react";
import { cn } from "@/lib/utils";

export interface EntityCardProps extends React.HTMLAttributes<HTMLElement> {
  as?: React.ElementType;
  isHighlighted?: boolean;
}

export function EntityCard({
  as: Component = "article",
  className,
  isHighlighted = false,
  children,
  ...props
}: EntityCardProps) {
  return (
    <Component
      className={cn("entity-card", isHighlighted && "is-highlighted", className)}
      {...props}
    >
      {children}
    </Component>
  );
}

export function EntityCardHeader({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("entity-card-header", className)} {...props}>
      {children}
    </div>
  );
}

export function EntityCardHeading({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("entity-card-heading", className)} {...props}>
      {children}
    </div>
  );
}

export function EntityCardTitle({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h2 className={cn("entity-card-title", className)} {...props}>
      {children}
    </h2>
  );
}

export function EntityCardEyebrow({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("entity-card-eyebrow", className)} {...props}>
      {children}
    </div>
  );
}

export function EntityCardSubtitle({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p className={cn("entity-card-subtitle", className)} {...props}>
      {children}
    </p>
  );
}

export function EntityCardAside({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("entity-card-aside", className)} {...props}>
      {children}
    </div>
  );
}

export function EntityCardBody({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("entity-card-body", className)} {...props}>
      {children}
    </div>
  );
}

export function EntityCardFooter({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("entity-card-footer", className)} {...props}>
      {children}
    </div>
  );
}

export function EntityCardMeta({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("entity-card-meta", className)} {...props}>
      {children}
    </div>
  );
}

export function EntityCardActions({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("entity-card-actions", className)} {...props}>
      {children}
    </div>
  );
}

EntityCard.Header = EntityCardHeader;
EntityCard.Heading = EntityCardHeading;
EntityCard.Title = EntityCardTitle;
EntityCard.Eyebrow = EntityCardEyebrow;
EntityCard.Subtitle = EntityCardSubtitle;
EntityCard.Aside = EntityCardAside;
EntityCard.Body = EntityCardBody;
EntityCard.Footer = EntityCardFooter;
EntityCard.Meta = EntityCardMeta;
EntityCard.Actions = EntityCardActions;
