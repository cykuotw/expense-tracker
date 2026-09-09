import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import NotificationSettings from "./NotificationSettings";

describe("NotificationSettings", () => {
    it("is limited to the mobile layout", () => {
        render(<NotificationSettings />);

        expect(screen.getByText("Activity notifications").closest("section")).toHaveClass(
            "md:hidden",
        );
    });
});
