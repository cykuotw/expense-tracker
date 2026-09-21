import { Component, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardFooter,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";

type AppErrorBoundaryProps = {
    children: ReactNode;
    onError?: () => void;
};

type AppErrorBoundaryState = {
    hasError: boolean;
};

class AppErrorBoundary extends Component<
    AppErrorBoundaryProps,
    AppErrorBoundaryState
> {
    state: AppErrorBoundaryState = { hasError: false };

    static getDerivedStateFromError(): AppErrorBoundaryState {
        return { hasError: true };
    }

    componentDidCatch(): void {
        this.props.onError?.();
    }

    private handleRetry = (): void => {
        this.setState({ hasError: false });
    };

    render(): ReactNode {
        if (!this.state.hasError) {
            return this.props.children;
        }

        return (
            <main className="flex min-h-screen min-h-dvh items-center justify-center bg-background px-4 py-10">
                <Card
                    className="w-full max-w-md"
                    role="alert"
                    aria-labelledby="app-error-title"
                    aria-describedby="app-error-description"
                >
                    <CardHeader>
                        <CardTitle>
                            <h1 id="app-error-title">
                                We couldn't display this page.
                            </h1>
                        </CardTitle>
                        <CardDescription id="app-error-description">
                            An unexpected problem stopped this screen from
                            loading.
                        </CardDescription>
                    </CardHeader>
                    <CardContent>
                        <p>Try again now, or go home and continue from there.</p>
                    </CardContent>
                    <CardFooter className="flex-col gap-2 sm:flex-row">
                        <Button
                            type="button"
                            className="min-h-11 w-full sm:w-auto"
                            onClick={this.handleRetry}
                        >
                            Try again
                        </Button>
                        <Button
                            asChild
                            variant="outline"
                            className="min-h-11 w-full sm:w-auto"
                        >
                            <a href="/">Go home</a>
                        </Button>
                    </CardFooter>
                </Card>
            </main>
        );
    }
}

export default AppErrorBoundary;
