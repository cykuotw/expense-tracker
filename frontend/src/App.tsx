import { BrowserRouter as Router, Routes, Route } from "react-router-dom";

import NavbarLayout from "./layouts/NavbarLayout";
import "./styles/styles.css";

function Home() {
    return (
        <>
            <h1 className="text-3xl font-bold underline text-red-500">
                Hello Tailwindcss!
            </h1>
            <div className="p-10">
                <button className="btn btn-secondary">DaisyUI Button</button>
            </div>
        </>
    );
}

function App() {
    return (
        <>
            <Router>
                <Routes>
                    <Route element={<NavbarLayout />}>
                        <Route path="/" element={<Home />} />
                    </Route>
                </Routes>
            </Router>
        </>
    );
}

export default App;
