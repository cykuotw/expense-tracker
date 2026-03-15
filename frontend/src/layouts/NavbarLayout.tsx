import { Outlet } from "react-router-dom";

import Navbar from "../components/navbar/Navbar";
import NavbarMobile from "../components/navbar/NavbarMobile";
import { UserRole } from "../types/role";

interface NavbarLayoutProps {
    role: UserRole | null;
}

export default function NavbarLayout({ role }: NavbarLayoutProps) {
    return (
        <div className="app-shell">
            <Navbar role={role} />
            <NavbarMobile role={role} />
            <main id="main-content" className="app-shell__content">
                <Outlet />
            </main>
        </div>
    );
}
