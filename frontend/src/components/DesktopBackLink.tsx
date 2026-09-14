import { mdiArrowLeft } from "@mdi/js";
import Icon from "@mdi/react";
import { Link } from "react-router-dom";

interface DesktopBackLinkProps {
    label: string;
    to: string;
}

export default function DesktopBackLink({ label, to }: DesktopBackLinkProps) {
    return (
        <div className="desktop-page-utility">
            <Link className="desktop-back-link" to={to}>
                <span className="desktop-back-link__icon" aria-hidden="true">
                    <Icon path={mdiArrowLeft} size={0.82} />
                </span>
                <span>{label}</span>
            </Link>
        </div>
    );
}
