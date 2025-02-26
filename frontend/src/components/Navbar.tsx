import { Link } from "react-router-dom";
import Dropdown from "./Dropdown";
import "../styles/styles.css";

export default function Navbar() {
    return (
        <>
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
                                    <Link to="create_group">
                                        Create New Group
                                    </Link>
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

            {/* small screen bottom navbar */}
            <div className="dock dock-xl bg-primary text-primary-content z-50 md:hidden">
                <button className="btn-wide">
                    <Link to="/">
                        <div className="flex items-center">
                            <svg
                                className="size-[1.8em] bi bi-people-fill"
                                xmlns="http://www.w3.org/2000/svg"
                                width="16"
                                height="16"
                                fill="currentColor"
                                viewBox="0 0 24 24"
                            >
                                <path d="M10,20V14H14V20H19V12H22L12,3L2,12H5V20H10Z"></path>
                            </svg>
                            Home
                        </div>
                    </Link>
                </button>

                <button className="btn-wide">
                    <Link to="/create_group">
                        <div className="flex items-center">
                            <svg
                                className="size-[1.8em] bi bi-people-fill"
                                xmlns="http://www.w3.org/2000/svg"
                                width="16"
                                height="16"
                                fill="currentColor"
                                viewBox="0 0 16 16"
                            >
                                <path d="M7 14s-1 0-1-1 1-4 5-4 5 3 5 4-1 1-1 1zm4-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6m-5.784 6A2.24 2.24 0 0 1 5 13c0-1.355.68-2.75 1.936-3.72A6.3 6.3 0 0 0 5 9c-4 0-5 3-5 4s1 1 1 1zM4.5 8a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5"></path>
                            </svg>
                            Create New Group
                        </div>
                    </Link>
                </button>
            </div>
        </>
    );
}
