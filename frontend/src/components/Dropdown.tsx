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
            <div role="button" className="flex items-center gap-1">
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
                    stroke="#ffffff"
                    strokeWidth="3"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                >
                    <path d="M6 9l6 6 6-6" />
                </svg>
            </div>

            {/* Dropdown Content */}
            <ul
                className={`dropdown-content menu bg-base-100 text-base-content ${contendTextConfig} z-10 my-3 space-y-2 py-3 shadow items-center absolute`}
            >
                {children}
            </ul>
        </div>
    );
};

export default Dropdown;
