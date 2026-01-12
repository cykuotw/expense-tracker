import { Link } from "react-router-dom";

interface NavbarMobileProps {
    role: string | null;
}

export default function NavbarMobile({ role }: NavbarMobileProps) {
    return (
        <div className="dock dock-xl bg-neutral/90 text-neutral-content backdrop-blur border-t border-base-300 z-50 md:hidden">
            <button className="btn-wide">
                <Link to="/">
                    <div className="flex flex-col items-center gap-1">
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
                        <span className="text-xs">Home</span>
                    </div>
                </Link>
            </button>

            <button className="btn-wide">
                <Link to="/create_group">
                    <div className="flex flex-col items-center gap-1">
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
                        <span className="text-xs">Create</span>
                    </div>
                </Link>
            </button>

            {role === "admin" && (
                <button className="btn-wide">
                    <Link to="/admin/invite">
                        <div className="flex flex-col items-center gap-1">
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                width="16"
                                height="16"
                                fill="currentColor"
                                className="size-[1.8em] bi bi-person-plus-fill"
                                viewBox="0 0 16 16"
                            >
                                <path d="M1 14s-1 0-1-1 1-4 6-4 6 3 6 4-1 1-1 1zm5-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6" />
                                <path
                                    fillRule="evenodd"
                                    d="M13.5 5a.5.5 0 0 1 .5.5V7h1.5a.5.5 0 0 1 0 1H14v1.5a.5.5 0 0 1-1 0V8h-1.5a.5.5 0 0 1 0-1H13V5.5a.5.5 0 0 1 .5-.5"
                                />
                            </svg>
                            <span className="text-xs">Invite</span>
                        </div>
                    </Link>
                </button>
            )}
        </div>
    );
}
