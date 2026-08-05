import { Button, type ButtonProps } from '@appica/ui-react/button'

export function AppButton({ className = '', ...props }: ButtonProps) {
  return <Button className={`gap-2 ${className}`} {...props} />
}
