import {
    BrowserRouter as Router,
    Routes,
    Route,
    useLocation,
} from "react-router-dom";
import { useState, useEffect } from "react";

import { API_URL } from "./configs/config";

import NavbarLayout from "./layouts/NavbarLayout";
import Login from "./pages/Login";
import Register from "./pages/Register";
import GuestGuard from "./components/auth/GuestGuard";
import AuthGuard from "./components/auth/AuthGuard";
import AdminGuard from "./components/auth/AdminGuard";
import Home from "./pages/Home";
import GroupDetail from "./pages/GroupDetail";
import AddMember from "./pages/AddMember";
import ExpenseDetail from "./pages/ExpenseDetail";
import CreateExpense from "./pages/CreateExpense";
import CreateGroup from "./pages/CreateGroup";
import EditExpense from "./pages/EditExpense";
import InviteUser from "./pages/InviteUser";

function AppRoutes() {
    const [authState, setAuthState] = useState<{
        isAuthenticated: boolean;
        role: string | null;
    } | null>(null);
    const location = useLocation();

    useEffect(() => {
        const checkAuth = async () => {
            try {
                const response = await fetch(`${API_URL}/auth/me`, {
                    method: "GET",
                    credentials: "include",
                });

                if (response.ok) {
                    const data = await response.json();
                    setAuthState({ isAuthenticated: true, role: data.role });
                } else {
                    setAuthState({ isAuthenticated: false, role: null });
                }
            } catch {
                setAuthState({ isAuthenticated: false, role: null });
            }
        };

        checkAuth();
    }, [location.pathname]);

    if (authState === null) {
        return (
            <div className="flex items-center justify-center h-screen">
                <span className="loading loading-spinner loading-xl"></span>
            </div>
        );
    }

    return (
        <Routes>
            <Route
                element={
                    <GuestGuard isAuthenticated={authState.isAuthenticated} />
                }
            >
                <Route path="/register" element={<Register />} />
                <Route path="/login" element={<Login />} />
            </Route>

            <Route
                element={
                    <AuthGuard isAuthenticated={authState.isAuthenticated} />
                }
            >
                <Route element={<NavbarLayout role={authState.role} />}>
                    <Route path="/" element={<Home />} />

                    <Route path="/group/:id" element={<GroupDetail />} />
                    <Route path="/create_group" element={<CreateGroup />} />

                    <Route path="/expense/:id" element={<ExpenseDetail />} />
                    <Route path="/expense/:id/edit" element={<EditExpense />} />
                    <Route path="/create_expense" element={<CreateExpense />} />
                    <Route path="/add_member" element={<AddMember />} />
                </Route>
            </Route>

            <Route
                element={
                    <AdminGuard
                        isAuthenticated={authState.isAuthenticated}
                        role={authState.role}
                    />
                }
            >
                <Route element={<NavbarLayout role={authState.role} />}>
                    <Route path="/admin/invite" element={<InviteUser />} />
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
