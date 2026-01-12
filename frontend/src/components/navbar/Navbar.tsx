import { Link } from "react-router-dom";

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
        <div className="hidden md:block sticky top-0 z-50 border-b border-base-300 bg-neutral/90 text-neutral-content backdrop-blur">
            <div className="navbar mx-auto w-full max-w-6xl px-4">
                <div className="navbar-start">
                    <Link to="/" className="btn btn-ghost text-xl">
                        Expense Tracker
                    </Link>
                </div>

                <div className="navbar-center hidden md:flex">
                    <ul className="menu menu-horizontal px-4 text-base">
                        {role === "admin" && (
                            <li>
                                <Link to="/admin/invite">Invite User</Link>
                            </li>
                        )}
                        <li>
                            <Dropdown label="Groups">
                                <li className="w-max">
                                    <Link to="/create_group">
                                        Create New Group
                                    </Link>
                                </li>
                            </Dropdown>
                        </li>
                    </ul>
                </div>

                <div className="navbar-end">
                    <ul className="menu menu-horizontal px-1">
                        <li>
                            <Dropdown
                                label={
                                    <div className="flex items-center text-sm gap-2">
                                        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-base-200 text-base-content/70">
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                width="18"
                                                height="18"
                                                fill="currentColor"
                                                viewBox="0 0 16 16"
                                            >
                                                <path d="M3 14s-1 0-1-1 1-4 6-4 6 3 6 4-1 1-1 1zm5-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6"></path>
                                            </svg>
                                        </div>
                                        <span>Account</span>
                                    </div>
                                }
                                dropdownType="dropdown-bottom dropdown-end"
                                contendTextConfig="text-sm"
                            >
                                <li>
                                    <button onClick={handleLogout}>
                                        Logout
                                    </button>
                                </li>
                            </Dropdown>
                        </li>
                    </ul>
                </div>
            </div>
        </div>
    );
}
