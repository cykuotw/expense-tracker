import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { mdiArrowLeft } from "@mdi/js";
import Icon from "@mdi/react";
import { useReturnNavigation } from "../hooks/navigation";

interface MobilePageHeaderProps {
    title: string;
    backTo?: string;
    backLabel?: string;
    titleIcon?: ReactNode;
    action?: ReactNode;
    backMode?: "return" | "destination";
}

function MobileReturnLink({ to, label, mode }: { to: string; label: string; mode: "return" | "destination" }) {
    const back = useReturnNavigation(to, label);
    return (
        <Link
            className="ui-button ui-button-outline mobile-page-header__back min-h-12 min-w-12 px-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            to={mode === "destination" ? to : back.to}
            onClick={mode === "destination" ? undefined : back.onClick}
            aria-label={mode === "destination" ? label : back.label}
        >
            <Icon path={mdiArrowLeft} size={1.1} aria-hidden="true" />
        </Link>
    );
}

export default function MobilePageHeader({
    title,
    backTo,
    backLabel = "Back",
    titleIcon,
    action,
    backMode = "return",
}: MobilePageHeaderProps) {
    return (
        <header className="page-header mobile-page-header mobile-page-header--sticky md:hidden">
            {backTo ? (
                <MobileReturnLink to={backTo} label={backLabel} mode={backMode} />
            ) : (
                <span aria-hidden="true" />
            )}
            <div className="mobile-page-header__title-wrap">
                {titleIcon}
                <h1 className="mobile-page-header__title" title={title}>
                    {title}
                </h1>
            </div>
            <div className="mobile-page-header__action">{action}</div>
        </header>
    );
}
