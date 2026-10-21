import { lazy, Suspense, type ReactNode } from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { Toaster } from "react-hot-toast";

import { AuthProvider } from "./contexts/AuthContext";
import { useAuth } from "./hooks/AuthContextHooks";
import NavbarLayout from "./layouts/NavbarLayout";
import RouteGuard from "./components/auth/RouteGuard";
import AppErrorBoundary from "./components/AppErrorBoundary";
import NavigationProvider from "./contexts/NavigationProvider";
import OfflineScreen from "./components/pwa/OfflineScreen";
import PWAUpdatePrompt from "./components/pwa/PWAUpdatePrompt";
import { PWAInstallProvider } from "./contexts/PWAInstallProvider";
import { PWAUpdateSafetyProvider } from "./contexts/PWAUpdateSafetyProvider";
import { reportFrontendRenderError } from "./lib/frontendErrorReporting";

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
const MonthlyReview = lazy(() => import("./pages/MonthlyReview"));
const ReceiptEditorLab = import.meta.env.DEV
    ? lazy(() => import("./pages/ReceiptEditorLab"))
    : null;

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
                <Route element={<RouteGuard mode="guest" />}>
                    <Route path="/register" element={<Register />} />
                    <Route path="/login" element={<Login />} />
                </Route>

                <Route element={<RouteGuard mode="authenticated" />}>
                    <Route element={<NavbarLayout />}>
                        <Route path="/" element={<Home />} />

                        <Route path="/group/:id" element={<GroupDetail />} />
                        <Route path="/group/:id/monthly-review/:month" element={<MonthlyReview />} />
                        <Route path="/create_group" element={<CreateGroup />} />
                        <Route path="/group/:id/edit" element={<EditGroup />} />

                        <Route
                            path="/expense/:id"
                            element={<ExpenseDetail />}
                        />
                        <Route
                            path="/expense/:id/edit/*"
                            element={<EditExpense />}
                        />
                        <Route
                            path="/create_expense/*"
                            element={<CreateExpense />}
                        />
                        <Route path="/add_member" element={<AddMember />} />
                        <Route path="/account" element={<AccountSettings />} />
                    </Route>
                </Route>

                <Route element={<RouteGuard mode="admin" />}>
                    <Route element={<NavbarLayout />}>
                        <Route path="/admin/users" element={<AdminUsers />} />
                    </Route>
                </Route>
            </Routes>
        </Suspense>
    );
}

function SessionNavigation({ children }: { children: ReactNode }) {
    const { userID, isAuthenticated } = useAuth();
    return <NavigationProvider key={`${isAuthenticated}:${userID ?? "guest"}`}>{children}</NavigationProvider>;
}

function App() {
    const receiptEditorLab = ReceiptEditorLab &&
        window.location.pathname === "/__dev/receipt-editor";

    return (
        <AppErrorBoundary onError={reportFrontendRenderError}>
            <Router>
                {receiptEditorLab ? (
                    <Suspense fallback={<RouteFallback />}>
                        <Routes>
                            <Route path="/__dev/receipt-editor" element={<ReceiptEditorLab />} />
                        </Routes>
                    </Suspense>
                ) : (
                    <>
                        <PWAInstallProvider>
                            <PWAUpdateSafetyProvider>
                                <AuthProvider>
                                    <SessionNavigation>
                                        <Toaster position="bottom-center" />
                                        {import.meta.env.PROD ? <PWAUpdatePrompt /> : null}
                                        <AppRoutes />
                                    </SessionNavigation>
                                </AuthProvider>
                            </PWAUpdateSafetyProvider>
                        </PWAInstallProvider>
                    </>
                )}
            </Router>
        </AppErrorBoundary>
    );
}

export default App;
