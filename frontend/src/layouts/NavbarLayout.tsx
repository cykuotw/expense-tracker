import { Outlet } from "react-router-dom";
import Navbar from "../components/Navbar";
import "../styles/styles.css";

export default function NavbarLayout() {
    return (
        <>
            <Navbar />
            <div id="main-content">
                <Outlet />
            </div>
        </>
    );
}
