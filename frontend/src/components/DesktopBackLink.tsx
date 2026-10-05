import { mdiArrowLeft } from "@mdi/js";
import Icon from "@mdi/react";
import { Link } from "react-router-dom";
import { useReturnNavigation } from "../hooks/navigation";

interface DesktopBackLinkProps {
    label: string;
    to: string;
    mode?: "return" | "destination";
}

export default function DesktopBackLink({ label, to, mode = "return" }: DesktopBackLinkProps) {
    const back = useReturnNavigation(to, label);
    const completion = mode === "destination";
    return (
        <div className="desktop-page-utility">
            <Link className="desktop-back-link" to={completion ? to : back.to} onClick={completion ? undefined : back.onClick}>
                <span className="desktop-back-link__icon" aria-hidden="true">
                    <Icon path={mdiArrowLeft} size={0.82} />
                </span>
                <span>{completion ? label : back.label}</span>
            </Link>
        </div>
    );
}
