import { useState, useRef, ReactNode } from "react";

interface DropdownProps {
    label: string | ReactNode;
    dropdownType?: string;
    contendTextConfig?: string;
    children: ReactNode;
}

const Dropdown: React.FC<DropdownProps> = ({
    label,
    dropdownType = "dropdown-center",
    contendTextConfig = "text-base w-max max-w-64 min-w-48",
    children,
}) => {
    const [isOpen, setIsOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    return (
        <div
            className={`dropdown ${dropdownType} relative`}
            ref={dropdownRef}
            tabIndex={0}
            onBlur={(e) => {
                if (!dropdownRef.current?.contains(e.relatedTarget as Node)) {
                    setIsOpen(false);
                }
            }}
            onClick={(e) => {
                e.stopPropagation();
                if (!isOpen) setIsOpen(true);
            }}
        >
            {/* Dropdown Button */}
            <button
                type="button"
                className="flex w-full items-center justify-between gap-2 rounded-2xl px-3 py-2 text-left text-current hover:bg-base-200/70"
                onClick={(e) => {
                    e.stopPropagation();
                    setIsOpen((prev) => !prev);
                }}
            >
                {label}
                <svg
                    className={`size-[1em] transition-transform duration-200 ${
                        isOpen ? "rotate-180" : "rotate-0"
                    }`}
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="3"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                >
                    <path d="M6 9l6 6 6-6" />
                </svg>
            </button>

            {/* Dropdown Content */}
            <ul
                className={`dropdown-content menu bg-base-100 text-base-content ${contendTextConfig} ${
                    isOpen ? "" : "hidden"
                } z-10 my-3 space-y-2 rounded-3xl py-3 shadow items-center absolute border border-base-300`}
            >
                {children}
            </ul>
        </div>
    );
};

export default Dropdown;
