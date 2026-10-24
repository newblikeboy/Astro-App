import { LoaderCircle, Info } from "lucide-react";
import { t } from "./i18n";
/** Button renders the shared styled button while forwarding standard button properties. */
export function Button({
  children,
  className = "",
  busy = false,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  busy?: boolean;
}) {
  return (
    <button
      className={"button " + className}
      {...props}
      disabled={props.disabled || busy}
    >
      {busy ? <LoaderCircle className="spin" size={18} /> : null}
      {children}
    </button>
  );
}
/** ErrorBox renders a readable, accessible error message only when an error is present. */
export function ErrorBox({ error }: { error: string }) {
  return error ? (
    <p className="error" role="alert">
      <Info size={17} />
      {t(error)}
    </p>
  ) : null;
}
