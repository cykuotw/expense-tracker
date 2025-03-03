import GoogleSignInButton from "../../components/auth/GoogleSignInButton";
import LoginForm from "../../components/auth/LoginForm";

import "../../styles/styles.css";

export default function Login() {
    return (
        <div className="h-screen flex flex-col justify-center items-center gap-y-2">
            <LoginForm />
            <p>or</p>
            <GoogleSignInButton />
            <div className="flex flex-row items-center gap-x-2">
                <p>New here?</p>
                <a href="/register" className="link link-info">
                    Create an account
                </a>
            </div>
        </div>
    );
}
