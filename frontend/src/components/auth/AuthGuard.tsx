import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../../hooks/AuthContextHooks";

const AuthGuard = () => {
    const { isAuthenticated } = useAuth();
    return isAuthenticated ? <Outlet /> : <Navigate to="/login" replace />;
};

export default AuthGuard;
