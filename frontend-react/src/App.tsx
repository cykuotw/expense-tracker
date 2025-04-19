import {
    BrowserRouter as Router,
    Routes,
    Route,
    useLocation,
} from "react-router-dom";
import { useState, useEffect } from "react";

import NavbarLayout from "./layouts/NavbarLayout";
import Login from "./pages/Login";
import Register from "./pages/Register";
import GuestGuard from "./components/auth/GuestGuard";
import AuthGuard from "./components/auth/AuthGuard";
import Home from "./pages/Home";
import GroupDetail from "./pages/GroupDetail";
import AddMember from "./pages/AddMember";
import ExpenseDetail from "./pages/ExpenseDetail";
import CreateExpense from "./pages/CreateExpense";
import CreateGroup from "./pages/CreateGroup";

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

    if (isAuthenticated === null) {
        return (
            <div className="flex items-center justify-center h-screen">
                <span className="loading loading-spinner loading-xl"></span>
            </div>
        );
    }

    return (
        <Routes>
            <Route element={<GuestGuard isAuthenticated={isAuthenticated} />}>
                <Route path="/register" element={<Register />} />
                <Route path="/login" element={<Login />} />
            </Route>

            <Route element={<AuthGuard isAuthenticated={isAuthenticated} />}>
                <Route element={<NavbarLayout />}>
                    <Route path="/" element={<Home />} />

                    <Route path="/group/:id" element={<GroupDetail />} />
                    <Route path="/create_group" element={<CreateGroup />} />

                    <Route path="/expense/:id" element={<ExpenseDetail />} />
                    <Route path="/create_expense" element={<CreateExpense />} />
                    <Route path="/add_member" element={<AddMember />} />
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
