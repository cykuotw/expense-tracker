import { Link } from "react-router-dom";

import Dropdown from "../Dropdown";

export default function Navbar() {
    return (
        <div className="navbar bg-neutral text-neutral-content hidden md:flex relative z-50">
            <div className="navbar-start">
                <a href="/" className="btn btn-ghost text-2xl">
                    Expense Tracker
                </a>
            </div>

            {/* middle to large screen top navbar */}
            <div className="navbar-center hidden md:flex">
                <ul className="menu menu-horizontal px-5 text-lg">
                    <li>
                        <Dropdown label="Group">
                            <li className="w-max">
                                <Link to="create_group">Create New Group</Link>
                            </li>
                        </Dropdown>
                    </li>
                </ul>
            </div>
            <div className="navbar-end">
                <div className="flex-none">
                    <ul className="menu menu-horizontal px-1">
                        <li>
                            <Dropdown
                                label={
                                    <div className="flex items-center text-lg gap-1">
                                        <svg
                                            xmlns="http://www.w3.org/2000/svg"
                                            width="22"
                                            height="22"
                                            fill="currentColor"
                                            className="bi bi-person-fill"
                                            viewBox="0 0 16 16"
                                        >
                                            <path d="M3 14s-1 0-1-1 1-4 6-4 6 3 6 4-1 1-1 1zm5-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6"></path>
                                        </svg>
                                        <p>Account</p>
                                    </div>
                                }
                                dropdownType="dropdown-bottom dropdown-end"
                                contendTextConfig="text-sm"
                            >
                                <li>
                                    <Link to="/profile">Profile</Link>
                                </li>
                                <li>
                                    <Link to="/settings">Settings</Link>
                                </li>
                                <li>
                                    <Link to="/logout">Logout</Link>
                                </li>
                            </Dropdown>
                        </li>
                    </ul>
                </div>
            </div>
        </div>
    );
}
