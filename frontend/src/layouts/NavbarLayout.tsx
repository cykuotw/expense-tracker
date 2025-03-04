import { Outlet } from "react-router-dom";

import Navbar from "../components/navbar/Navbar";
import NavbarMobile from "../components/navbar/NavbarMobile";

export default function NavbarLayout() {
    return (
        <>
            <Navbar />
            <NavbarMobile />
            <div id="main-content">
                <Outlet />
            </div>
        </>
    );
}
