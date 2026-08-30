import { useState } from "react";
import { Link, NavLink } from "react-router-dom";

import { useAuth } from "../../hooks/AuthContextHooks";
import { USER_ROLES } from "../../types/role";
import InstallAppAction from "../pwa/InstallAppAction";

export default function NavbarMobile() {
    const [accountOpen, setAccountOpen] = useState(false);
    const { role, logout } = useAuth();

    return (
        <>
            <nav className="app-shell__mobile-nav md:hidden" aria-label="Primary">
            <div className="app-shell__mobile-nav-inner">
                <NavLink
                    to="/"
                    className={({ isActive }) =>
                        `app-shell__mobile-item ${
                            isActive ? "app-shell__mobile-item--active" : ""
                        }`
                    }
                >
                    <svg
                        className="size-[1.5rem]"
                        xmlns="http://www.w3.org/2000/svg"
                        width="16"
                        height="16"
                        fill="currentColor"
                        viewBox="0 0 24 24"
                    >
                        <path d="M10,20V14H14V20H19V12H22L12,3L2,12H5V20H10Z"></path>
                    </svg>
                    <span className="app-shell__mobile-item-label">Home</span>
                </NavLink>

                <button
                    type="button"
                    className={`app-shell__mobile-item ${
                        accountOpen ? "app-shell__mobile-item--active" : ""
                    }`}
                    onClick={() => setAccountOpen(true)}
                    aria-label="Open account actions"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="16"
                        height="16"
                        fill="currentColor"
                        className="size-[1.5rem]"
                        viewBox="0 0 16 16"
                    >
                        <path d="M3 14s-1 0-1-1 1-4 6-4 6 3 6 4-1 1-1 1zm5-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6"></path>
                    </svg>
                    <span className="app-shell__mobile-item-label">
                        Account
                    </span>
                </button>

            </div>
            </nav>

            <div
                className={`fixed inset-0 z-[60] bg-foreground/20 transition-opacity md:hidden ${
                    accountOpen
                        ? "pointer-events-auto opacity-100"
                        : "pointer-events-none opacity-0"
                }`}
                onClick={() => setAccountOpen(false)}
                aria-hidden={!accountOpen}
            >
                <div
                    className={`absolute inset-x-0 bottom-0 rounded-t-[2rem] border border-border bg-background p-6 shadow-2xl transition-transform ${
                        accountOpen ? "translate-y-0" : "translate-y-full"
                    }`}
                    style={{
                        paddingBottom: "calc(1.5rem + env(safe-area-inset-bottom))",
                    }}
                    onClick={(e) => e.stopPropagation()}
                >
                    <div className="mx-auto mb-5 h-1.5 w-14 rounded-full bg-border" />
                    <div className="section-label">Account</div>
                    <div className="mt-2 text-lg font-semibold text-foreground">
                        Account actions
                    </div>
                    <div className="mt-6 space-y-3">
                        <Link
                            to="/account"
                            className="ui-button ui-button-primary w-full"
                            onClick={() => setAccountOpen(false)}
                        >
                            Settings
                        </Link>
                        <InstallAppAction
                            className="ui-button ui-button-outline w-full"
                            onOpen={() => setAccountOpen(false)}
                        />
                        {role === USER_ROLES.admin ? (
                            <Link
                                to="/admin/users"
                                className="ui-button ui-button-outline w-full"
                                onClick={() => setAccountOpen(false)}
                            >
                                User Management
                            </Link>
                        ) : null}
                        <button
                            type="button"
                            className="ui-button ui-button-destructive w-full"
                            onClick={() => void logout()}
                        >
                            Logout
                        </button>
                        <button
                            type="button"
                            className="ui-button ui-button-ghost w-full"
                            onClick={() => setAccountOpen(false)}
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            </div>
        </>
    );
}
