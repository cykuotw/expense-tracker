import { Navigate, Outlet } from "react-router-dom";

const AuthGuard = ({ isAuthenticated }: { isAuthenticated: boolean }) => {
    return isAuthenticated ? <Outlet /> : <Navigate to="/login" replace />;
};

export default AuthGuard;
