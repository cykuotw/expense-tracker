import { Link, NavLink } from "react-router-dom";

import Dropdown from "../Dropdown";
import { API_URL } from "../../configs/config";

interface NavbarProps {
    role: string | null;
}

export default function Navbar({ role }: NavbarProps) {
    const handleLogout = async () => {
        await fetch(`${API_URL}/logout`, {
            method: "POST",
            credentials: "include",
        });
        window.location.href = "/login";
    };

    return (
        <aside className="app-shell__desktop-nav">
            <div className="panel-card flex w-full flex-col rounded-[2rem] p-5">
                <div className="border-b border-base-300/70 pb-5">
                    <Link to="/" className="block rounded-2xl p-3">
                        <div className="text-xs font-bold uppercase tracking-[0.24em] text-primary/70">
                            Personal Finance
                        </div>
                        <div className="mt-2 text-2xl font-bold tracking-[-0.04em] text-base-content">
                            Expense Tracker
                        </div>
                    </Link>
                </div>

                <div className="mt-6">
                    <div className="section-label px-3">Workspace</div>
                    <ul className="mt-3 flex flex-col gap-2">
                        <li>
                            <NavLink
                                to="/"
                                className={({ isActive }) =>
                                    `flex items-center rounded-2xl px-4 py-3 text-sm font-medium ${
                                        isActive
                                            ? "bg-primary/10 text-primary"
                                            : "text-base-content/70 hover:bg-base-200/80"
                                    }`
                                }
                            >
                                Overview
                            </NavLink>
                        </li>
                        <li>
                            <NavLink
                                to="/create_group"
                                className={({ isActive }) =>
                                    `flex items-center rounded-2xl px-4 py-3 text-sm font-medium ${
                                        isActive
                                            ? "bg-primary/10 text-primary"
                                            : "text-base-content/70 hover:bg-base-200/80"
                                    }`
                                }
                            >
                                Create Group
                            </NavLink>
                        </li>
                        {role === "admin" && (
                            <li>
                                <NavLink
                                    to="/admin/invite"
                                    className={({ isActive }) =>
                                        `flex items-center rounded-2xl px-4 py-3 text-sm font-medium ${
                                            isActive
                                                ? "bg-primary/10 text-primary"
                                                : "text-base-content/70 hover:bg-base-200/80"
                                        }`
                                    }
                                >
                                    Invite Users
                                </NavLink>
                            </li>
                        )}
                    </ul>
                </div>

                <div className="mt-auto rounded-[1.5rem] bg-base-200/70 p-4">
                    <div className="flex items-center gap-3">
                        <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                width="20"
                                height="20"
                                fill="currentColor"
                                viewBox="0 0 16 16"
                            >
                                <path d="M3 14s-1 0-1-1 1-4 6-4 6 3 6 4-1 1-1 1zm5-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6"></path>
                            </svg>
                        </div>
                        <div>
                            <div className="text-sm font-semibold text-base-content">
                                Account
                            </div>
                            <div className="text-xs uppercase tracking-[0.18em] text-base-content/55">
                                {role ?? "member"}
                            </div>
                        </div>
                    </div>
                    <div className="mt-4">
                        <Dropdown
                            label="Actions"
                            dropdownType="dropdown-top"
                            contendTextConfig="text-sm w-48"
                        >
                            <li className="w-full">
                                <button onClick={handleLogout}>Logout</button>
                            </li>
                        </Dropdown>
                    </div>
                </div>
            </div>
        </aside>
    );
}
