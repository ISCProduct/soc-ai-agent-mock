"use client"
import * as React from "react"
import { X } from "lucide-react"

interface DialogProps extends React.HTMLAttributes<HTMLDivElement> {
  open: boolean
  onOpenChange: (open: boolean) => void
  children: React.ReactNode
}

export function Dialog({ open, onOpenChange, children }: DialogProps) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={() => onOpenChange(false)} />
      <div className="relative z-50 w-full max-w-5xl mx-auto">{children}</div>
    </div>
  )
}

export function DialogContent({ className = "", children }: { className?: string; children: React.ReactNode }) {
  return (
    <div className={`bg-background border rounded-lg shadow-lg p-6 ${className}`}> {children} </div>
  )
}

export function DialogHeader({ children }: { children: React.ReactNode }) {
  return <div className="mb-4">{children}</div>
}

export function DialogTitle({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <h2 className={`text-xl font-bold ${className}`}>{children}</h2>
}
