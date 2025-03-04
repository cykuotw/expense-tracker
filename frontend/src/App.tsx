import {
    BrowserRouter as Router,
    Routes,
    Route,
    useLocation,
} from "react-router-dom";
import { useState, useEffect } from "react";

import "./styles/styles.css";

import NavbarLayout from "./layouts/NavbarLayout";
import Login from "./pages/auth/Login";
import Register from "./pages/auth/Register";
import GuestGuard from "./components/auth/GuestGuard";
import AuthGuard from "./components/auth/AuthGuard";

function Home() {
    return (
        <>
            <h1 className="text-3xl font-bold underline text-red-500">
                Hello Tailwindcss!
            </h1>
            <div className="p-10">
                <button className="btn btn-secondary">DaisyUI Button</button>
            </div>
        </>
    );
}

function AppRoutes() {
    const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(
        null
    );
    const location = useLocation();

    useEffect(() => {
        const checkAuth = async () => {
            try {
                const response = await fetch(
                    "http://localhost:8080/api/v0/auth/me",
                    {
                        method: "GET",
                        credentials: "include",
                    }
                );

                setIsAuthenticated(response.ok);
            } catch {
                setIsAuthenticated(false);
            }
        };

        checkAuth();
    }, [location.pathname]);

    if (isAuthenticated === null)
        return <span className="loading loading-spinner loading-md"></span>;

    return (
        <Routes>
            <Route element={<GuestGuard isAuthenticated={isAuthenticated} />}>
                <Route path="/register" element={<Register />} />
                <Route path="/login" element={<Login />} />
            </Route>

            <Route element={<AuthGuard isAuthenticated={isAuthenticated} />}>
                <Route element={<NavbarLayout />}>
                    <Route path="/" element={<Home />} />
                </Route>
            </Route>
        </Routes>
    );
}

function App() {
    return (
        <Router>
            <AppRoutes />
        </Router>
    );
}

export default App;
