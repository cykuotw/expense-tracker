import { ExpenseAllocationMode } from "../../types/allocation";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/tabs";

const MODE_OPTIONS: Array<{
    mode: ExpenseAllocationMode;
    label: string;
    description: string;
}> = [
    {
        mode: "equal",
        label: "Equally",
        description: "Share the total evenly across selected people.",
    },
    {
        mode: "exact",
        label: "Exact amounts",
        description: "Enter the final amount each person owes.",
    },
    {
        mode: "percentage",
        label: "Percentage",
        description: "Assign percentages that add up to 100%.",
    },
    {
        mode: "adjustment",
        label: "Equal + adjustments",
        description: "Split the remainder equally after adding extras.",
    },
];

interface ExpenseAllocationModeSelectorProps {
    mode: ExpenseAllocationMode;
    onModeChange: (mode: ExpenseAllocationMode) => void;
}

export default function ExpenseAllocationModeSelector({
    mode,
    onModeChange,
}: ExpenseAllocationModeSelectorProps) {
    return (
        <>
            <Tabs
                className="md:hidden"
                value={mode}
                onValueChange={(value) =>
                    onModeChange(value as ExpenseAllocationMode)
                }
            >
                <div className="w-full overflow-x-auto pb-1">
                    <TabsList
                        aria-label="Allocation mode"
                        className="w-max min-w-full justify-start"
                    >
                        {MODE_OPTIONS.map((option) => (
                            <TabsTrigger
                                className="shrink-0"
                                key={option.mode}
                                value={option.mode}
                            >
                                {option.label}
                            </TabsTrigger>
                        ))}
                    </TabsList>
                </div>
                {MODE_OPTIONS.map((option) => (
                    <TabsContent
                        className="pt-3 text-sm leading-5 text-muted-foreground"
                        key={option.mode}
                        value={option.mode}
                    >
                        {option.description}
                    </TabsContent>
                ))}
            </Tabs>

            <fieldset className="hidden gap-3 md:grid md:grid-cols-2">
                <legend className="sr-only">Allocation mode</legend>
                {MODE_OPTIONS.map((option) => (
                    <label
                        className="flex min-h-20 cursor-pointer items-start gap-3 rounded-xl border border-border p-4 transition-colors hover:bg-muted/50 has-[:checked]:border-primary has-[:checked]:bg-primary/5 focus-within:ring-2 focus-within:ring-primary"
                        key={option.mode}
                    >
                        <input
                            className="mt-1 size-4 accent-primary"
                            type="radio"
                            name="allocation-mode"
                            value={option.mode}
                            checked={mode === option.mode}
                            onChange={() => onModeChange(option.mode)}
                        />
                        <span className="min-w-0">
                            <span className="block font-semibold">
                                {option.label}
                            </span>
                            <span className="mt-1 block text-sm leading-5 text-muted-foreground">
                                {option.description}
                            </span>
                        </span>
                    </label>
                ))}
            </fieldset>
        </>
    );
}
