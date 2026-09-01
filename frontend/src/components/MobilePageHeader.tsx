import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { mdiArrowLeft } from "@mdi/js";
import Icon from "@mdi/react";

interface MobilePageHeaderProps {
    title: string;
    backTo?: string;
    backLabel?: string;
    titleIcon?: ReactNode;
    action?: ReactNode;
}

export default function MobilePageHeader({
    title,
    backTo,
    backLabel = "Back",
    titleIcon,
    action,
}: MobilePageHeaderProps) {
    return (
        <header className="page-header mobile-page-header md:hidden">
            {backTo ? (
                <Link
                    className="ui-button ui-button-outline mobile-page-header__back min-h-12 min-w-12 px-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                    to={backTo}
                    aria-label={backLabel}
                >
                    <Icon path={mdiArrowLeft} size={1.1} aria-hidden="true" />
                </Link>
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
