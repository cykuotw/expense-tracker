import {
    BrowserRouter as Router,
    Routes,
    Route,
} from "react-router-dom";
import { Toaster } from "react-hot-toast";

import { AuthProvider } from "./contexts/AuthContext";
import { useAuth } from "./hooks/AuthContextHooks";
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
    const { loading } = useAuth();

    if (loading) {
        return (
            <div className="flex items-center justify-center h-screen">
                <span className="loading loading-spinner loading-xl"></span>
            </div>
        );
    }

    return (
        <Routes>
            <Route element={<GuestGuard />}>
                <Route path="/register" element={<Register />} />
                <Route path="/login" element={<Login />} />
            </Route>

            <Route element={<AuthGuard />}>
                <Route element={<NavbarLayout />}>
                    <Route path="/" element={<Home />} />

                    <Route path="/group/:id" element={<GroupDetail />} />
                    <Route path="/create_group" element={<CreateGroup />} />

                    <Route path="/expense/:id" element={<ExpenseDetail />} />
                    <Route path="/expense/:id/edit" element={<EditExpense />} />
                    <Route path="/create_expense" element={<CreateExpense />} />
                    <Route path="/add_member" element={<AddMember />} />
                </Route>
            </Route>

            <Route element={<AdminGuard />}>
                <Route element={<NavbarLayout />}>
                    <Route path="/admin/invite" element={<InviteUser />} />
                </Route>
            </Route>
        </Routes>
    );
}

function App() {
    return (
        <Router>
            <AuthProvider>
                <Toaster position="bottom-center" />
                <AppRoutes />
            </AuthProvider>
        </Router>
    );
}

export default App;
