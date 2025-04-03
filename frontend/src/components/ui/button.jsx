import * as React from "react"
import { Slot } from "@radix-ui/react-slot"

const Button = React.forwardRef(
  ({ className, variant = "default", size = "default", asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button"
    
    const variantClass = variant === "default" ? "button-default"
      : variant === "destructive" ? "button-destructive"
      : variant === "outline" ? "button-outline"
      : variant === "secondary" ? "button-secondary"
      : variant === "ghost" ? "button-ghost"
      : variant === "link" ? "button-link"
      : "";
    
    const sizeClass = size === "default" ? "button-default-size"
      : size === "sm" ? "button-sm"
      : size === "lg" ? "button-lg"
      : size === "xs" ? "button-xs"
      : "";
    
    return (
      <Comp
        className={`button ${variantClass} ${sizeClass} ${className || ""}`}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button } 