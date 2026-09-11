import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import {
    createMemoryRouter,
    MemoryRouter,
    Route,
    RouterProvider,
    Routes,
} from "react-router-dom";
import SplitExpensePage from "./SplitExpensePage";

const { toastErrorMock } = vi.hoisted(() => ({ toastErrorMock: vi.fn() }));

vi.mock("react-hot-toast", () => ({
    toast: { error: toastErrorMock },
}));

const members = [
    { userId: "user-a", username: "Alice" },
    { userId: "user-b", username: "Bob" },
];

beforeAll(() => {
    vi.stubGlobal(
        "ResizeObserver",
        class {
            observe() {}
            unobserve() {}
            disconnect() {}
        }
    );
});

function renderSplit({
    mainFormVisited = true,
    onSave = vi.fn(),
} = {}) {
    render(
        <MemoryRouter initialEntries={["/expense-form/split"]}>
            <Routes>
                <Route path="/expense-form" element={<p>Expense form</p>} />
                <Route
                    path="/expense-form/split"
                    element={
                        <SplitExpensePage
                            allocation={{
                                mode: "equal",
                                participants: members.map(({ userId }) => ({
                                    userId,
                                })),
                            }}
                            amountDigits={2}
                            currency="CAD"
                            groupMembers={members}
                            mainFormVisited={mainFormVisited}
                            onSave={onSave}
                            returnTo="/expense-form"
                            total="10.00"
                        />
                    }
                />
            </Routes>
        </MemoryRouter>
    );
    return onSave;
}

describe("SplitExpensePage", () => {
    afterEach(() => {
        cleanup();
        vi.clearAllMocks();
    });

    it("restores a mode's draft after switching away and saves only the active mode", () => {
        const onSave = renderSplit();
        fireEvent.click(screen.getByRole("radio", { name: /Exact amounts/ }));
        fireEvent.change(screen.getByLabelText("Alice amount (CAD)"), {
            target: { value: "6.00" },
        });
        fireEvent.change(screen.getByLabelText("Bob amount (CAD)"), {
            target: { value: "4.00" },
        });

        fireEvent.click(screen.getByRole("radio", { name: /Percentage/ }));
        fireEvent.click(screen.getByRole("radio", { name: /Exact amounts/ }));

        expect(screen.getByLabelText("Alice amount (CAD)")).toHaveValue(6);
        expect(screen.getByLabelText("Bob amount (CAD)")).toHaveValue(4);
        fireEvent.click(screen.getByRole("button", { name: "Save split" }));

        expect(onSave).toHaveBeenCalledWith({
            mode: "exact",
            participants: [
                { userId: "user-a", amount: "6.00" },
                { userId: "user-b", amount: "4.00" },
            ],
        });
        expect(screen.getByText("Expense form")).toBeInTheDocument();
    });

    it("keeps cancel changes local to the split page", () => {
        const onSave = renderSplit();
        fireEvent.click(screen.getByRole("radio", { name: /Exact amounts/ }));
        fireEvent.change(screen.getByLabelText("Alice amount (CAD)"), {
            target: { value: "9.00" },
        });
        fireEvent.click(
            screen.getByRole("link", { name: "Cancel split changes" })
        );

        expect(onSave).not.toHaveBeenCalled();
        expect(screen.getByText("Expense form")).toBeInTheDocument();
    });

    it("saves the participant selection instead of relying on zero values", () => {
        const onSave = renderSplit();

        fireEvent.click(
            screen.getByRole("checkbox", { name: "Include Bob" }),
        );
        fireEvent.click(screen.getByRole("button", { name: "Save split" }));

        expect(onSave).toHaveBeenCalledWith({
            mode: "equal",
            participants: [{ userId: "user-a" }],
        });
    });

    it("discards the local draft when browser history navigates back", async () => {
        const onSave = vi.fn();
        const router = createMemoryRouter(
            [
                { path: "/expense-form", element: <p>Expense form</p> },
                {
                    path: "/expense-form/split",
                    element: (
                        <SplitExpensePage
                            allocation={{
                                mode: "equal",
                                participants: members.map(({ userId }) => ({
                                    userId,
                                })),
                            }}
                            amountDigits={2}
                            currency="CAD"
                            groupMembers={members}
                            mainFormVisited
                            onSave={onSave}
                            returnTo="/expense-form"
                            total="10.00"
                        />
                    ),
                },
            ],
            {
                initialEntries: ["/expense-form", "/expense-form/split"],
                initialIndex: 1,
            },
        );
        render(<RouterProvider router={router} />);
        fireEvent.click(screen.getByRole("radio", { name: /Exact amounts/ }));
        fireEvent.change(screen.getByLabelText("Alice amount (CAD)"), {
            target: { value: "9.00" },
        });

        await act(() => router.navigate(-1));

        expect(screen.getByText("Expense form")).toBeInTheDocument();
        expect(onSave).not.toHaveBeenCalled();
    });

    it("focuses an error summary and reveals inline errors after failed save", () => {
        renderSplit();
        fireEvent.click(screen.getByRole("radio", { name: /Percentage/ }));
        fireEvent.change(screen.getByLabelText("Alice percentage (%)"), {
            target: { value: "33.333" },
        });
        fireEvent.click(screen.getByRole("button", { name: "Save split" }));

        const alert = screen.getByRole("alert");
        expect(alert).toHaveFocus();
        expect(screen.getByLabelText(/^Alice percentage \(%\)/)).toHaveAttribute(
            "aria-invalid",
            "true"
        );
        expect(
            screen.getByText(/at most two decimal places/, {
                selector: "span",
            })
        ).toBeInTheDocument();
    });

    it("redirects a direct split-page load to recover the parent draft", async () => {
        renderSplit({ mainFormVisited: false });

        expect(screen.getByText("Expense form")).toBeInTheDocument();
        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "Open the expense form before configuring its split.",
                { id: "expense-split-direct-route-recovery" }
            );
        });
    });

    it("places the mobile save action in the navigation-safe action bar", () => {
        renderSplit();

        expect(
            screen.getByRole("button", { name: "Save split" }).parentElement
        ).toHaveClass("expense-split-actions");
    });

    it("exposes mobile allocation tabs with the selected rule description", () => {
        renderSplit();

        expect(screen.getByRole("tab", { name: "Equally" })).toHaveAttribute(
            "aria-selected",
            "true"
        );
        fireEvent.mouseDown(screen.getByRole("tab", { name: "Percentage" }), {
            button: 0,
            ctrlKey: false,
        });

        expect(screen.getByRole("tabpanel")).toHaveTextContent(
            "Assign percentages that add up to 100%."
        );
        expect(screen.getByRole("tab", { name: "Percentage" })).toHaveAttribute(
            "aria-selected",
            "true"
        );
        expect(screen.getByLabelText("Alice percentage (%)")).toBeInTheDocument();
    });

    it("groups split method and participants into one mobile surface", () => {
        renderSplit();

        const editor = screen
            .getByText("How should it be split?")
            .closest('[data-slot="split-editor"]');

        expect(editor).not.toBeNull();
        expect(editor).toContainElement(screen.getByText("Participants"));
        expect(editor).toHaveClass("rounded-xl", "md:contents");
    });

    it("uses a compact form grid with shared column headers", () => {
        renderSplit();
        fireEvent.click(screen.getByRole("radio", { name: /Exact amounts/ }));

        const header = document.querySelector(
            '[data-slot="split-participant-header"]'
        );
        const aliceRow = screen
            .getByRole("checkbox", { name: "Include Alice" })
            .closest('[data-slot="split-participant"]');

        expect(header).toHaveTextContent("Person");
        expect(header).toHaveTextContent("Amount (CAD)");
        expect(aliceRow).toHaveClass(
            "grid-cols-[minmax(0,1fr)_minmax(7.5rem,0.85fr)]",
            "px-3",
            "py-2"
        );
        expect(screen.queryByText(/^Final share:/)).not.toBeInTheDocument();
        expect(
            aliceRow?.querySelector('[id="split-exact-user-a-error"]')
        ).toBeNull();
    });
});
