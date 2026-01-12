import GoogleSignInButton from "../components/auth/GoogleSignInButton";
import LoginForm from "../components/auth/LoginForm";
import { LoginProvider } from "../contexts/LoginContext";

export default function Login() {
    return (
        <div className="h-screen flex flex-col justify-center items-center gap-y-2">
            <LoginProvider>
                <LoginForm />
            </LoginProvider>

            <p>or</p>

            <GoogleSignInButton />
        </div>
    );
}
