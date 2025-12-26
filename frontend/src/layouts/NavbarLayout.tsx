import { Outlet } from "react-router-dom";

import Navbar from "../components/navbar/Navbar";
import NavbarMobile from "../components/navbar/NavbarMobile";

interface NavbarLayoutProps {
    role: string | null;
}

export default function NavbarLayout({ role }: NavbarLayoutProps) {
    return (
        <>
            <Navbar role={role} />
            <NavbarMobile role={role} />
            <div id="main-content">
                <Outlet />
            </div>
        </>
    );
}
