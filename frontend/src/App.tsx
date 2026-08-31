import { lazy, Suspense } from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { Toaster } from "react-hot-toast";

import { AuthProvider } from "./contexts/AuthContext";
import { useAuth } from "./hooks/AuthContextHooks";
import NavbarLayout from "./layouts/NavbarLayout";
import GuestGuard from "./components/auth/GuestGuard";
import AuthGuard from "./components/auth/AuthGuard";
import AdminGuard from "./components/auth/AdminGuard";
import MobileScrollToTop from "./components/MobileScrollToTop";
import OfflineScreen from "./components/pwa/OfflineScreen";
import PWAUpdatePrompt from "./components/pwa/PWAUpdatePrompt";
import { PWAInstallProvider } from "./contexts/PWAInstallProvider";

const Login = lazy(() => import("./pages/Login"));
const Register = lazy(() => import("./pages/Register"));
const Home = lazy(() => import("./pages/Home"));
const GroupDetail = lazy(() => import("./pages/GroupDetail"));
const AddMember = lazy(() => import("./pages/AddMember"));
const ExpenseDetail = lazy(() => import("./pages/ExpenseDetail"));
const CreateExpense = lazy(() => import("./pages/CreateExpense"));
const CreateGroup = lazy(() => import("./pages/CreateGroup"));
const EditGroup = lazy(() => import("./pages/EditGroup"));
const EditExpense = lazy(() => import("./pages/EditExpense"));
const AdminUsers = lazy(() => import("./pages/AdminUsers"));
const AccountSettings = lazy(() => import("./pages/AccountSettings"));

function RouteFallback() {
    return (
        <div
            className="flex h-screen items-center justify-center"
            role="status"
        >
            <span className="ui-spinner ui-spinner-xl" aria-hidden="true" />
            <span className="sr-only">Loading page</span>
        </div>
    );
}

function AppRoutes() {
    const { loading, isOffline } = useAuth();

    if (loading) {
        return (
            <div className="flex items-center justify-center h-screen">
                <span className="ui-spinner ui-spinner-xl"></span>
            </div>
        );
    }

    if (isOffline) {
        return <OfflineScreen />;
    }

    return (
        <Suspense fallback={<RouteFallback />}>
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
                        <Route path="/group/:id/edit" element={<EditGroup />} />

                        <Route
                            path="/expense/:id"
                            element={<ExpenseDetail />}
                        />
                        <Route
                            path="/expense/:id/edit"
                            element={<EditExpense />}
                        />
                        <Route
                            path="/create_expense"
                            element={<CreateExpense />}
                        />
                        <Route path="/add_member" element={<AddMember />} />
                        <Route path="/account" element={<AccountSettings />} />
                    </Route>
                </Route>

                <Route element={<AdminGuard />}>
                    <Route element={<NavbarLayout />}>
                        <Route path="/admin/users" element={<AdminUsers />} />
                    </Route>
                </Route>
            </Routes>
        </Suspense>
    );
}

function App() {
    return (
        <Router>
            <MobileScrollToTop />
            <PWAInstallProvider>
                <AuthProvider>
                    <Toaster position="bottom-center" />
                    {import.meta.env.PROD ? <PWAUpdatePrompt /> : null}
                    <AppRoutes />
                </AuthProvider>
            </PWAInstallProvider>
        </Router>
    );
}

export default App;
