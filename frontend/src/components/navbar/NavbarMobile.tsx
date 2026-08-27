import { useState } from "react";
import { NavLink } from "react-router-dom";

import { useAuth } from "../../hooks/AuthContextHooks";
import { USER_ROLES } from "../../types/role";

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

                {role === USER_ROLES.admin && (
                    <NavLink
                        to="/admin/users"
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
                            viewBox="0 0 24 24"
                        >
                            <path d="M16 13c-.29 0-.62.02-.97.05C16.19 13.89 17 15.24 17 17v2H7v-2c0-1.76.81-3.11 1.97-3.95C8.62 13.02 8.29 13 8 13c-2.67 0-8 1.34-8 4v3h24v-3c0-2.66-5.33-4-8-4M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4Z" />
                        </svg>
                        <span className="app-shell__mobile-item-label">
                            Users
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
                            onClick={() => void logout()}
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
