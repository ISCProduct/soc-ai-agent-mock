"use client"
import * as React from "react"

interface TabsProps {
  value: string
  onValueChange: (val: string) => void
  children: React.ReactNode
  className?: string
}

export function Tabs({ value, onValueChange, children, className = "" }: TabsProps) {
  return <div className={className}>{children}</div>
}

export function TabsList({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <div className={`flex gap-2 ${className}`}>{children}</div>
}

export function TabsTrigger({ value, children }: { value: string; children: React.ReactNode }) {
  return (
    <button
      type="button"
      onClick={() => (document.dispatchEvent(new CustomEvent("tabs-trigger", { detail: { value } }))) }
      className="px-3 py-2 text-sm font-medium rounded-md bg-muted hover:bg-muted/70"
    >
      {children}
    </button>
  )
}

export function TabsContent({ value, children, className = "" }: { value: string; children: React.ReactNode; className?: string }) {
  return <div data-tabs-content={value} className={className}>{children}</div>
}
