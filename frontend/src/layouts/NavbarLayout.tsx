import { Outlet } from "react-router-dom";

import Navbar from "../components/navbar/Navbar";
import NavbarMobile from "../components/navbar/NavbarMobile";

export default function NavbarLayout() {
    return (
        <div className="app-shell">
            <Navbar />
            <NavbarMobile />
            <main id="main-content" className="app-shell__content">
                <Outlet />
            </main>
        </div>
    );
}
