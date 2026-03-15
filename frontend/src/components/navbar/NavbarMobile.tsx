import { useState } from "react";
import { NavLink } from "react-router-dom";

import { API_URL } from "../../configs/config";

interface NavbarMobileProps {
    role: string | null;
}

export default function NavbarMobile({ role }: NavbarMobileProps) {
    const [accountOpen, setAccountOpen] = useState(false);

    const handleLogout = async () => {
        await fetch(`${API_URL}/logout`, {
            method: "POST",
            credentials: "include",
        });
        window.location.href = "/login";
    };

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

                <NavLink
                    to="/create_group"
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
                        viewBox="0 0 16 16"
                    >
                        <path d="M7 14s-1 0-1-1 1-4 5-4 5 3 5 4-1 1-1 1zm4-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6m-5.784 6A2.24 2.24 0 0 1 5 13c0-1.355.68-2.75 1.936-3.72A6.3 6.3 0 0 0 5 9c-4 0-5 3-5 4s1 1 1 1zM4.5 8a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5"></path>
                    </svg>
                    <span className="app-shell__mobile-item-label">
                        Create
                    </span>
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

                {role === "admin" && (
                    <NavLink
                        to="/admin/invite"
                        className={({ isActive }) =>
                            `app-shell__mobile-item ${
                                isActive
                                    ? "app-shell__mobile-item--active"
                                    : ""
                            }`
                        }
                    >
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            width="16"
                            height="16"
                            fill="currentColor"
                            className="size-[1.5rem]"
                            viewBox="0 0 16 16"
                        >
                            <path d="M1 14s-1 0-1-1 1-4 6-4 6 3 6 4-1 1-1 1zm5-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6" />
                            <path
                                fillRule="evenodd"
                                d="M13.5 5a.5.5 0 0 1 .5.5V7h1.5a.5.5 0 0 1 0 1H14v1.5a.5.5 0 0 1-1 0V8h-1.5a.5.5 0 0 1 0-1H13V5.5a.5.5 0 0 1 .5-.5"
                            />
                        </svg>
                        <span className="app-shell__mobile-item-label">
                            Invite
                        </span>
                    </NavLink>
                )}
            </div>
            </nav>

            <div
                className={`fixed inset-0 z-[60] bg-neutral/20 transition-opacity md:hidden ${
                    accountOpen
                        ? "pointer-events-auto opacity-100"
                        : "pointer-events-none opacity-0"
                }`}
                onClick={() => setAccountOpen(false)}
                aria-hidden={!accountOpen}
            >
                <div
                    className={`absolute inset-x-0 bottom-0 rounded-t-[2rem] border border-base-300 bg-base-100 p-6 shadow-2xl transition-transform ${
                        accountOpen ? "translate-y-0" : "translate-y-full"
                    }`}
                    style={{
                        paddingBottom: "calc(1.5rem + env(safe-area-inset-bottom))",
                    }}
                    onClick={(e) => e.stopPropagation()}
                >
                    <div className="mx-auto mb-5 h-1.5 w-14 rounded-full bg-base-300" />
                    <div className="section-label">Account</div>
                    <div className="mt-2 text-lg font-semibold text-base-content">
                        Account actions
                    </div>
                    <div className="mt-6 space-y-3">
                        <button
                            type="button"
                            className="btn btn-error w-full"
                            onClick={handleLogout}
                        >
                            Logout
                        </button>
                        <button
                            type="button"
                            className="btn btn-ghost w-full"
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
