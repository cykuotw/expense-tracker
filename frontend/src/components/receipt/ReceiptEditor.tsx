import {
    useCallback,
    useEffect,
    useLayoutEffect,
    useMemo,
    useRef,
    useState,
    type ChangeEvent,
    type PointerEvent as ReactPointerEvent,
} from "react";
import Icon from "@mdi/react";
import {
    mdiCrop,
    mdiFitToScreenOutline,
    mdiHandBackLeftOutline,
    mdiMagnifyMinusOutline,
    mdiMagnifyPlusOutline,
    mdiRectangleOutline,
    mdiRedo,
    mdiRestart,
    mdiRotateRight,
    mdiUndo,
} from "@mdi/js";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Spinner } from "@/components/ui/spinner";
import {
    commitReceiptEdit,
    createReceiptHistory,
    decodeReceiptFile,
    formatReceiptBytes,
    normalizeRect,
    orientedDimensions,
    prepareReceipt,
    redoReceiptEdit,
    rectFromPoints,
    renderReceiptPreview,
    rotateEditStateClockwise,
    undoReceiptEdit,
    updateRect,
    type DecodedReceipt,
    type NormalizedRect,
    type PreparedReceipt,
    type ReceiptEditState,
    type ReceiptMask,
} from "@/lib/receiptEditor";

type EditorMode = "crop" | "mask" | "pan";
type EditorPhase = "empty" | "loading" | "editing" | "exporting" | "error";

interface ReceiptEditorProps {
    onPrepared?: (receipt: PreparedReceipt) => void;
    onCancel?: () => void;
    onManualEntry?: () => void;
    initialFile?: File | null;
    onPhotoLoaded?: () => void;
    onPhotoCleared?: () => void;
    hideLibraryChoice?: boolean;
}

interface Point {
    x: number;
    y: number;
}

interface MaskDrag {
    kind: "move" | "resize";
    pointerId: number;
    start: Point;
    mask: ReceiptMask;
    state: ReceiptEditState;
}

const percent = (value: number) => `${value * 100}%`;
const percentValue = (value: number) => Math.round(value * 100);
const newMaskID = () => crypto.randomUUID();

function rectStyle(rect: NormalizedRect) {
    return {
        left: percent(rect.x),
        top: percent(rect.y),
        width: percent(rect.width),
        height: percent(rect.height),
    };
}

function replaceMask(state: ReceiptEditState, replacement: ReceiptMask): ReceiptEditState {
    return {
        ...state,
        masks: state.masks.map((mask) => mask.id === replacement.id ? replacement : mask),
    };
}

function RectFields({
    legend,
    rect,
    onChange,
}: {
    legend: string;
    rect: NormalizedRect;
    onChange: (rect: NormalizedRect) => void;
}) {
    const fields: Array<{ key: keyof NormalizedRect; label: string }> = [
        { key: "x", label: "Left" },
        { key: "y", label: "Top" },
        { key: "width", label: "Width" },
        { key: "height", label: "Height" },
    ];
    return (
        <fieldset className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <legend className="sr-only">{legend}</legend>
            {fields.map(({ key, label }) => (
                <label key={key} className="grid gap-1 text-xs font-medium text-muted-foreground">
                    {label} %
                    <input
                        className="min-h-11 rounded-xl border border-border bg-background px-3 text-base text-foreground"
                        type="number"
                        min={0}
                        max={100}
                        step={1}
                        value={percentValue(rect[key])}
                        onChange={(event) => {
                            const value = Number(event.target.value) / 100;
                            if (!Number.isFinite(value)) return;
                            onChange(updateRect(rect, { [key]: value }));
                        }}
                        aria-label={`${legend} ${label.toLowerCase()} percentage`}
                    />
                </label>
            ))}
        </fieldset>
    );
}

export default function ReceiptEditor({ onPrepared, onCancel, onManualEntry, initialFile, onPhotoLoaded, onPhotoCleared, hideLibraryChoice = false }: ReceiptEditorProps) {
    const [phase, setPhase] = useState<EditorPhase>("empty");
    const [decoded, setDecoded] = useState<DecodedReceipt | null>(null);
    const [history, setHistory] = useState(createReceiptHistory);
    const [mode, setMode] = useState<EditorMode>("crop");
    const [preciseControlsOpen, setPreciseControlsOpen] = useState(false);
    const [selectedMaskID, setSelectedMaskID] = useState<string | null>(null);
    const [draftRect, setDraftRect] = useState<NormalizedRect | null>(null);
    const [draftEdit, setDraftEdit] = useState<ReceiptEditState | null>(null);
    const [zoom, setZoom] = useState(1);
    const [pan, setPan] = useState({ x: 0, y: 0 });
    const [fitSize, setFitSize] = useState<{ width: number; height: number } | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [status, setStatus] = useState("Choose a receipt photo to begin.");
    const previewCanvasRef = useRef<HTMLCanvasElement>(null);
    const viewportRef = useRef<HTMLDivElement>(null);
    const stageRef = useRef<HTMLDivElement>(null);
    const drawStartRef = useRef<{ pointerId: number; point: Point } | null>(null);
    const maskDragRef = useRef<MaskDrag | null>(null);
    const panDragRef = useRef<{ pointerId: number; client: Point; pan: Point } | null>(null);
    const decodeControllerRef = useRef<AbortController | null>(null);
    const exportControllerRef = useRef<AbortController | null>(null);

    const edit = draftEdit ?? history.present;
    const selectedMask = edit.masks.find((mask) => mask.id === selectedMaskID) ?? null;
    const dimensions = decoded
        ? orientedDimensions(decoded.width, decoded.height, edit.rotation)
        : { width: 1, height: 1 };
    const viewIsFit = zoom === 1 && pan.x === 0 && pan.y === 0;

    const disposeSession = useCallback(() => {
        decodeControllerRef.current?.abort();
        exportControllerRef.current?.abort();
        setDecoded((current) => {
            current?.dispose();
            return null;
        });
        setHistory(createReceiptHistory());
        setDraftEdit(null);
        setDraftRect(null);
        setSelectedMaskID(null);
        setPreciseControlsOpen(false);
        setZoom(1);
        setPan({ x: 0, y: 0 });
    }, []);

    useEffect(() => disposeSession, [disposeSession]);

    useEffect(() => {
        if (!decoded || !previewCanvasRef.current || (phase !== "editing" && phase !== "exporting")) return;
        try {
            renderReceiptPreview(previewCanvasRef.current, decoded, edit.rotation);
        } catch (previewError) {
            setError(previewError instanceof Error ? previewError.message : "The preview could not be drawn.");
            setPhase("error");
        }
    }, [decoded, edit.rotation, phase]);

    useLayoutEffect(() => {
        const viewport = viewportRef.current;
        if (!decoded || !viewport || (phase !== "editing" && phase !== "exporting")) {
            setFitSize(null);
            return;
        }

        const updateFitSize = () => {
            const availableWidth = viewport.clientWidth;
            const availableHeight = Math.min(window.innerHeight * 0.7, 672);
            if (availableWidth <= 0 || availableHeight <= 0) return;
            const scale = Math.min(
                availableWidth / dimensions.width,
                availableHeight / dimensions.height,
            );
            setFitSize({
                width: Math.max(1, Math.round(dimensions.width * scale)),
                height: Math.max(1, Math.round(dimensions.height * scale)),
            });
        };

        updateFitSize();
        const observer = typeof ResizeObserver === "undefined"
            ? null
            : new ResizeObserver(updateFitSize);
        observer?.observe(viewport);
        window.addEventListener("resize", updateFitSize);
        return () => {
            observer?.disconnect();
            window.removeEventListener("resize", updateFitSize);
        };
    }, [decoded, dimensions.height, dimensions.width, phase]);

    const commit = useCallback((next: ReceiptEditState, message: string) => {
        setHistory((current) => commitReceiptEdit(current, next));
        setDraftEdit(null);
        setStatus(message);
    }, []);

    const loadFile = useCallback(async (file: File) => {
        disposeSession();
        const controller = new AbortController();
        decodeControllerRef.current = controller;
        setPhase("loading");
        setError(null);
        setStatus("Opening receipt photo…");
        try {
            const nextDecoded = await decodeReceiptFile(file, controller.signal);
            if (controller.signal.aborted) {
                nextDecoded.dispose();
                return;
            }
            setDecoded(nextDecoded);
            setHistory(createReceiptHistory());
            setPhase("editing");
            onPhotoLoaded?.();
            setStatus(`Receipt opened at ${nextDecoded.width} by ${nextDecoded.height} pixels.`);
        } catch (loadError) {
            if (loadError instanceof DOMException && loadError.name === "AbortError") return;
            setError(loadError instanceof Error ? loadError.message : "The photo could not be opened.");
            setPhase("error");
            setStatus("Receipt photo could not be opened.");
        }
    }, [disposeSession, onPhotoLoaded]);

    useEffect(() => {
        if (initialFile) void loadFile(initialFile);
    }, [initialFile, loadFile]);

    const handleFile = (event: ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        event.target.value = "";
        if (file) void loadFile(file);
    };

    const stagePoint = (event: ReactPointerEvent): Point | null => {
        const stage = stageRef.current;
        if (!stage) return null;
        const bounds = stage.getBoundingClientRect();
        if (bounds.width <= 0 || bounds.height <= 0) return null;
        return {
            x: Math.min(1, Math.max(0, (event.clientX - bounds.left) / bounds.width)),
            y: Math.min(1, Math.max(0, (event.clientY - bounds.top) / bounds.height)),
        };
    };

    const handleStagePointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
        if (mode === "pan") {
            panDragRef.current = {
                pointerId: event.pointerId,
                client: { x: event.clientX, y: event.clientY },
                pan,
            };
            event.currentTarget.setPointerCapture(event.pointerId);
            return;
        }
        const point = stagePoint(event);
        if (!point) return;
        drawStartRef.current = { pointerId: event.pointerId, point };
        setDraftRect({ x: point.x, y: point.y, width: 0.01, height: 0.01 });
        event.currentTarget.setPointerCapture(event.pointerId);
    };

    const handleStagePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
        const panDrag = panDragRef.current;
        if (panDrag?.pointerId === event.pointerId) {
            setPan({
                x: panDrag.pan.x + event.clientX - panDrag.client.x,
                y: panDrag.pan.y + event.clientY - panDrag.client.y,
            });
            return;
        }
        const drawStart = drawStartRef.current;
        if (drawStart?.pointerId !== event.pointerId) return;
        const point = stagePoint(event);
        if (point) setDraftRect(rectFromPoints(drawStart.point, point));
    };

    const finishStagePointer = (event: ReactPointerEvent<HTMLDivElement>) => {
        if (panDragRef.current?.pointerId === event.pointerId) {
            panDragRef.current = null;
            setStatus("Preview position changed. Export crop is unchanged.");
            return;
        }
        if (drawStartRef.current?.pointerId !== event.pointerId) return;
        drawStartRef.current = null;
        if (!draftRect || draftRect.width < 0.02 || draftRect.height < 0.02) {
            setDraftRect(null);
            return;
        }
        if (mode === "crop") {
            commit({ ...history.present, crop: normalizeRect(draftRect) }, "Crop area updated.");
        } else {
            const mask = { ...normalizeRect(draftRect), id: newMaskID() };
            commit({ ...history.present, masks: [...history.present.masks, mask] }, "Sensitive area masked.");
            setSelectedMaskID(mask.id);
        }
        setDraftRect(null);
    };

    const startMaskDrag = (
        event: ReactPointerEvent<HTMLElement>,
        mask: ReceiptMask,
        kind: MaskDrag["kind"],
    ) => {
        event.stopPropagation();
        const point = stagePoint(event);
        if (!point) return;
        setSelectedMaskID(mask.id);
        maskDragRef.current = {
            kind,
            pointerId: event.pointerId,
            start: point,
            mask,
            state: history.present,
        };
        event.currentTarget.setPointerCapture(event.pointerId);
    };

    const moveMaskDrag = (event: ReactPointerEvent<HTMLElement>) => {
        const drag = maskDragRef.current;
        if (!drag || drag.pointerId !== event.pointerId) return;
        const point = stagePoint(event);
        if (!point) return;
        const dx = point.x - drag.start.x;
        const dy = point.y - drag.start.y;
        const nextMask = drag.kind === "move"
            ? { ...drag.mask, ...updateRect(drag.mask, { x: drag.mask.x + dx, y: drag.mask.y + dy }) }
            : { ...drag.mask, ...updateRect(drag.mask, { width: drag.mask.width + dx, height: drag.mask.height + dy }) };
        setDraftEdit(replaceMask(drag.state, nextMask));
    };

    const finishMaskDrag = (event: ReactPointerEvent<HTMLElement>) => {
        const drag = maskDragRef.current;
        if (!drag || drag.pointerId !== event.pointerId) return;
        maskDragRef.current = null;
        if (draftEdit) commit(draftEdit, drag.kind === "move" ? "Mask moved." : "Mask resized.");
    };

    const exportReceipt = async () => {
        if (!decoded) return;
        const controller = new AbortController();
        exportControllerRef.current = controller;
        setPhase("exporting");
        setError(null);
        setStatus("Flattening edits and compressing receipt…");
        try {
            const result = await prepareReceipt(decoded, history.present, controller.signal);
            if (controller.signal.aborted) return;
            setPhase("editing");
            setStatus(`Prepared receipt is ${formatReceiptBytes(result.blob.size)}. Scanning now.`);
            onPrepared?.(result);
        } catch (exportError) {
            if (exportError instanceof DOMException && exportError.name === "AbortError") {
                setPhase("editing");
                setStatus("Receipt preparation cancelled.");
                return;
            }
            setError(exportError instanceof Error ? exportError.message : "The receipt could not be prepared.");
            setPhase("error");
            setStatus("Receipt preparation failed.");
        }
    };

    const reset = () => {
        setHistory(createReceiptHistory());
        setSelectedMaskID(null);
        setDraftEdit(null);
        setZoom(1);
        setPan({ x: 0, y: 0 });
        setStatus("All receipt edits were reset.");
    };

    const fitPreview = () => {
        setZoom(1);
        setPan({ x: 0, y: 0 });
        setStatus("Preview fitted to the available space. Export crop is unchanged.");
    };

    const rotateReceipt = () => {
        commit(rotateEditStateClockwise(history.present), "Receipt rotated 90 degrees and preview fitted.");
        setZoom(1);
        setPan({ x: 0, y: 0 });
    };

    const startOver = () => {
        disposeSession();
        setPhase("empty");
        setError(null);
        onPhotoCleared?.();
        setStatus("Choose a receipt photo to begin.");
    };

    const cancel = () => {
        startOver();
        onCancel?.();
    };

    const cancelExport = () => {
        exportControllerRef.current?.abort();
        setPhase("editing");
        setStatus("Receipt preparation cancelled. Your edits are still available.");
    };

    const manualEntry = () => {
        disposeSession();
        setPhase("empty");
        setError(null);
        setStatus("Continuing with manual entry.");
        onManualEntry?.();
    };

    const commitCrop = (crop: NormalizedRect) => commit({ ...history.present, crop }, "Crop area updated.");
    const commitSelectedMask = (mask: ReceiptMask) => commit(replaceMask(history.present, mask), "Mask updated.");

    const editorAspect = useMemo(
        () => `${dimensions.width} / ${dimensions.height}`,
        [dimensions.height, dimensions.width],
    );

    if (phase === "empty" || phase === "loading") {
        return (
            <section className="rounded-[2rem] border border-border bg-card p-5 shadow-[var(--card-shadow-soft)] sm:p-7 lg:mx-auto lg:max-w-3xl" aria-labelledby="receipt-editor-title">
                <div className="grid gap-2">
                    <div className="section-label">Receipt preparation</div>
                    <h2 id="receipt-editor-title" className="text-2xl font-bold text-foreground">Prepare a receipt photo</h2>
                    <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
                        Crop the receipt and permanently cover anything you do not want to send. The original photo stays in this browser session.
                    </p>
                </div>
                <div className="mt-5 grid gap-3 sm:grid-cols-2">
                    <Button asChild className="min-h-12 cursor-pointer text-base">
                        <label>
                            Take photo
                            <input className="sr-only" type="file" accept="image/*" capture="environment" onChange={handleFile} disabled={phase === "loading"} />
                        </label>
                    </Button>
                    {!hideLibraryChoice ? <Button asChild variant="outline" className="min-h-12 cursor-pointer text-base">
                        <label>
                            Choose photo
                            <input className="sr-only" type="file" accept="image/*" onChange={handleFile} disabled={phase === "loading"} />
                        </label>
                    </Button> : null}
                </div>
                {phase === "loading" ? (
                    <div className="mt-4 flex min-h-12 items-center gap-3 rounded-2xl bg-muted px-4 text-sm" role="status">
                        <Spinner aria-hidden="true" /> Opening photo…
                        <Button type="button" variant="ghost" className="ml-auto min-h-11" onClick={cancel}>Cancel</Button>
                    </div>
                ) : null}
                <Button type="button" variant="link" className="mt-3 min-h-11 px-0" onClick={manualEntry}>Continue with manual entry</Button>
                <p className="sr-only" aria-live="polite">{status}</p>
            </section>
        );
    }

    if (phase === "error") {
        return (
            <section className="rounded-[2rem] border border-destructive/30 bg-card p-5 sm:p-7" aria-labelledby="receipt-error-title">
                <div className="section-label text-destructive">Receipt preparation</div>
                <h2 id="receipt-error-title" className="mt-2 text-xl font-bold">We could not prepare this photo</h2>
                <p className="mt-2 text-sm leading-6 text-muted-foreground" role="alert">{error}</p>
                <div className="mt-5 flex flex-wrap gap-3">
                    <Button type="button" className="min-h-11" onClick={startOver}>Choose another photo</Button>
                    {decoded ? <Button type="button" variant="outline" className="min-h-11" onClick={() => setPhase("editing")}>Return to editor</Button> : null}
                    <Button type="button" variant="ghost" className="min-h-11" onClick={manualEntry}>Manual entry</Button>
                </div>
            </section>
        );
    }

    return (
        <section className="rounded-[2rem] border border-border bg-card p-4 shadow-[var(--card-shadow-soft)] sm:p-6" aria-labelledby="receipt-edit-title" aria-busy={phase === "exporting"}>
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div>
                    <div className="section-label">Receipt preparation</div>
                    <h2 id="receipt-edit-title" className="mt-1 text-2xl font-bold">Prepare your receipt</h2>
                    <p className="mt-1 text-sm text-muted-foreground">Crop the edges, rotate if needed, and mask anything private.</p>
                </div>
                <div className="hidden flex-wrap gap-2 md:flex" role="toolbar" aria-label="Receipt editing history">
                    <Button type="button" variant="outline" className="min-h-11" disabled={history.past.length === 0 || phase === "exporting"} onClick={() => { setHistory(undoReceiptEdit); setStatus("Last edit undone."); }}>Undo</Button>
                    <Button type="button" variant="outline" className="min-h-11" disabled={history.future.length === 0 || phase === "exporting"} onClick={() => { setHistory(redoReceiptEdit); setStatus("Edit restored."); }}>Redo</Button>
                    <Button type="button" variant="ghost" className="min-h-11" disabled={phase === "exporting"} onClick={reset}>Reset</Button>
                </div>
                <div className="grid w-fit grid-cols-3 gap-1 rounded-2xl border border-border bg-background p-1 md:hidden" role="toolbar" aria-label="Receipt editing history" data-testid="mobile-receipt-history">
                    <Button type="button" variant="ghost" size="icon" className="size-11" aria-label="Undo" disabled={history.past.length === 0 || phase === "exporting"} onClick={() => { setHistory(undoReceiptEdit); setStatus("Last edit undone."); }}><Icon path={mdiUndo} size={0.9} aria-hidden="true" /></Button>
                    <Button type="button" variant="ghost" size="icon" className="size-11" aria-label="Redo" disabled={history.future.length === 0 || phase === "exporting"} onClick={() => { setHistory(redoReceiptEdit); setStatus("Edit restored."); }}><Icon path={mdiRedo} size={0.9} aria-hidden="true" /></Button>
                    <Button type="button" variant="ghost" size="icon" className="size-11" aria-label="Reset receipt edits" disabled={phase === "exporting"} onClick={reset}><Icon path={mdiRestart} size={0.9} aria-hidden="true" /></Button>
                </div>
            </div>

            <div className="mt-5 hidden flex-wrap gap-2 md:flex" role="toolbar" aria-label="Receipt editing tools">
                {(["crop", "mask", "pan"] as const).map((tool) => (
                    <Button key={tool} type="button" variant={mode === tool ? "secondary" : "outline"} className="min-h-11 capitalize" aria-pressed={mode === tool} onClick={() => { setMode(tool); setStatus(`${tool} tool selected.`); }}>
                        {tool === "mask" ? "Mask area" : tool === "pan" ? "Pan preview" : "Crop"}
                    </Button>
                ))}
                <Button type="button" variant="outline" className="min-h-11" onClick={rotateReceipt}>Rotate 90°</Button>
                <Button type="button" variant="outline" className="min-h-11" disabled={zoom <= 1} onClick={() => setZoom((value) => Math.max(1, value - 0.25))} aria-label="Zoom out preview">Zoom −</Button>
                <Button type="button" variant="outline" className="min-h-11" disabled={zoom >= 3} onClick={() => setZoom((value) => Math.min(3, value + 0.25))} aria-label="Zoom in preview">Zoom +</Button>
                <Button type="button" variant="ghost" className="min-h-11" disabled={viewIsFit} onClick={fitPreview}>Fit preview</Button>
            </div>
            <div className="mt-5 grid gap-2 md:hidden" data-testid="mobile-receipt-toolbar">
                <div className="grid grid-cols-3 gap-1 rounded-2xl border border-border bg-muted/60 p-1" role="toolbar" aria-label="Receipt editing modes">
                    {([
                        { mode: "crop", label: "Crop", icon: mdiCrop },
                        { mode: "mask", label: "Mask", icon: mdiRectangleOutline },
                        { mode: "pan", label: "Pan", icon: mdiHandBackLeftOutline },
                    ] as const).map((tool) => (
                        <Button key={tool.mode} type="button" variant={mode === tool.mode ? "secondary" : "ghost"} className="min-h-12 flex-col gap-0.5 px-2 text-xs" aria-label={`${tool.label} mode`} aria-pressed={mode === tool.mode} onClick={() => { setMode(tool.mode); setStatus(`${tool.label} tool selected.`); }}>
                            <Icon path={tool.icon} size={0.78} aria-hidden="true" />
                            {tool.label}
                        </Button>
                    ))}
                </div>
                <div className="grid grid-cols-4 gap-2" role="toolbar" aria-label="Receipt preview actions">
                    <Button type="button" variant="outline" size="icon" className="size-11 w-full" aria-label="Rotate receipt 90 degrees" onClick={rotateReceipt}><Icon path={mdiRotateRight} size={0.9} aria-hidden="true" /></Button>
                    <Button type="button" variant="outline" size="icon" className="size-11 w-full" disabled={zoom <= 1} aria-label="Zoom out preview" onClick={() => setZoom((value) => Math.max(1, value - 0.25))}><Icon path={mdiMagnifyMinusOutline} size={0.9} aria-hidden="true" /></Button>
                    <Button type="button" variant="outline" size="icon" className="size-11 w-full" disabled={zoom >= 3} aria-label="Zoom in preview" onClick={() => setZoom((value) => Math.min(3, value + 0.25))}><Icon path={mdiMagnifyPlusOutline} size={0.9} aria-hidden="true" /></Button>
                    <Button type="button" variant="outline" size="icon" className="size-11 w-full" disabled={viewIsFit} aria-label="Fit preview" onClick={fitPreview}><Icon path={mdiFitToScreenOutline} size={0.9} aria-hidden="true" /></Button>
                </div>
            </div>

            <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_21rem] lg:items-start xl:grid-cols-[minmax(0,1fr)_24rem]">
                <div className="relative min-h-[18rem] min-w-0 overflow-hidden rounded-2xl border border-border bg-[#282824] p-3 sm:p-5 lg:min-h-[34rem]" style={{ touchAction: "none" }}>
                    <div ref={viewportRef} className="grid h-full min-h-[16rem] place-items-center overflow-hidden lg:min-h-[32rem]" data-testid="receipt-stage-viewport">
                        <div
                            ref={stageRef}
                            className="relative max-h-[70dvh] max-w-full origin-center select-none shadow-2xl"
                            style={{
                                aspectRatio: editorAspect,
                                width: fitSize ? `${fitSize.width}px` : "100%",
                                height: fitSize ? `${fitSize.height}px` : "auto",
                                transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
                                cursor: mode === "pan" ? "grab" : "crosshair",
                            }}
                            onPointerDown={handleStagePointerDown}
                            onPointerMove={handleStagePointerMove}
                            onPointerUp={finishStagePointer}
                            onPointerCancel={finishStagePointer}
                        >
                            <canvas ref={previewCanvasRef} className="block h-full w-full bg-white" role="img" aria-label="Receipt editing canvas" />
                            <div className="pointer-events-none absolute border-2 border-accent" style={{ ...rectStyle(edit.crop), boxShadow: "0 0 0 9999px rgb(0 0 0 / 0.58)" }} aria-hidden="true" />
                            {edit.masks.map((mask) => (
                                <div
                                    key={mask.id}
                                    className={`pointer-events-none absolute border-2 bg-[#111] ${selectedMaskID === mask.id ? "border-white ring-2 ring-accent" : "border-[#111]"}`}
                                    style={rectStyle(mask)}
                                >
                                    <button
                                        type="button"
                                        className="pointer-events-auto absolute left-1/2 top-1/2 size-11 -translate-x-1/2 -translate-y-1/2 cursor-move rounded-full bg-transparent"
                                        aria-label={`Mask ${edit.masks.indexOf(mask) + 1}. Drag to move.`}
                                        onClick={(event) => { event.stopPropagation(); setSelectedMaskID(mask.id); }}
                                        onPointerDown={(event) => startMaskDrag(event, mask, "move")}
                                        onPointerMove={moveMaskDrag}
                                        onPointerUp={finishMaskDrag}
                                        onPointerCancel={finishMaskDrag}
                                    />
                                    <button
                                        type="button"
                                        className="pointer-events-auto absolute -bottom-[22px] -right-[22px] grid size-11 cursor-se-resize place-items-center rounded-full bg-transparent"
                                        aria-label={`Resize mask ${edit.masks.indexOf(mask) + 1}`}
                                        onPointerDown={(event) => { event.stopPropagation(); startMaskDrag(event, mask, "resize"); }}
                                        onPointerMove={moveMaskDrag}
                                        onPointerUp={finishMaskDrag}
                                        onPointerCancel={finishMaskDrag}
                                    >
                                        <span aria-hidden="true" className="size-6 rounded-full border-2 border-white bg-accent" />
                                    </button>
                                </div>
                            ))}
                            {draftRect ? <div className={`pointer-events-none absolute border-2 ${mode === "mask" ? "border-white bg-[#111]" : "border-accent bg-accent/10"}`} style={rectStyle(draftRect)} /> : null}
                        </div>
                    </div>
                    <div className="pointer-events-none absolute bottom-3 left-3 rounded-full bg-black/70 px-3 py-1 text-xs font-medium text-white" data-testid="receipt-viewport-status">{viewIsFit ? "Fit" : `${Math.round(zoom * 100)}% zoom`}</div>
                </div>

                <aside className="grid min-w-0 content-start gap-3 lg:rounded-2xl lg:border lg:border-border lg:bg-background/60 lg:p-4" aria-label="Precise receipt controls">
                    <div className="hidden lg:block">
                        <h3 className="font-semibold">Adjustments</h3>
                        <p className="mt-1 text-sm text-muted-foreground">Fine-tune the crop and cover private details.</p>
                    </div>
                    <Button
                        type="button"
                        variant="outline"
                        className="min-h-12 w-full justify-between px-4 lg:hidden"
                        aria-expanded={preciseControlsOpen}
                        aria-controls="receipt-precise-controls"
                        onClick={() => setPreciseControlsOpen((open) => !open)}
                    >
                        <span>Precise controls</span>
                        <span className="text-xs font-normal text-muted-foreground">{preciseControlsOpen ? "Hide" : "Show"}</span>
                    </Button>
                    <div id="receipt-precise-controls" className={cn("gap-4 md:grid-cols-2 lg:grid-cols-1", preciseControlsOpen ? "grid" : "hidden lg:grid")}>
                        <div className="rounded-2xl border border-border bg-background p-4">
                            <h3 className="font-semibold">Precise crop</h3>
                            <p className="mt-1 text-xs leading-5 text-muted-foreground">Use these fields when dragging is difficult.</p>
                            <div className="mt-3"><RectFields legend="Crop" rect={history.present.crop} onChange={commitCrop} /></div>
                        </div>
                        <div className="rounded-2xl border border-border bg-background p-4">
                            <div className="flex items-center justify-between gap-3">
                                <div><h3 className="font-semibold">Masks</h3><p className="mt-1 text-xs text-muted-foreground">{history.present.masks.length} permanent region{history.present.masks.length === 1 ? "" : "s"}</p></div>
                                <Button type="button" size="sm" variant="outline" className="min-h-11" onClick={() => { const mask = { id: newMaskID(), x: 0.35, y: 0.45, width: 0.3, height: 0.1 }; commit({ ...history.present, masks: [...history.present.masks, mask] }, "Mask added in the center."); setSelectedMaskID(mask.id); }}>Add mask</Button>
                            </div>
                            {history.present.masks.length > 0 ? (
                                <div className="mt-3 flex flex-wrap gap-2" aria-label="Select a mask">
                                    {history.present.masks.map((mask, index) => (
                                        <Button
                                            key={mask.id}
                                            type="button"
                                            variant={selectedMaskID === mask.id ? "secondary" : "outline"}
                                            className="min-h-11"
                                            aria-pressed={selectedMaskID === mask.id}
                                            onClick={() => setSelectedMaskID(mask.id)}
                                        >
                                            Mask {index + 1}
                                        </Button>
                                    ))}
                                </div>
                            ) : null}
                            {selectedMask ? (
                                <div className="mt-3 grid gap-3">
                                    <RectFields legend="Selected mask" rect={selectedMask} onChange={(rect) => commitSelectedMask({ ...selectedMask, ...rect })} />
                                    <div className="grid grid-cols-3 gap-2" aria-label="Move selected mask">
                                        <span />
                                        <Button type="button" variant="outline" className="min-h-11" aria-label="Move mask up" onClick={() => commitSelectedMask({ ...selectedMask, ...updateRect(selectedMask, { y: selectedMask.y - 0.01 }) })}>↑</Button>
                                        <span />
                                        <Button type="button" variant="outline" className="min-h-11" aria-label="Move mask left" onClick={() => commitSelectedMask({ ...selectedMask, ...updateRect(selectedMask, { x: selectedMask.x - 0.01 }) })}>←</Button>
                                        <Button type="button" variant="outline" className="min-h-11" aria-label="Move mask down" onClick={() => commitSelectedMask({ ...selectedMask, ...updateRect(selectedMask, { y: selectedMask.y + 0.01 }) })}>↓</Button>
                                        <Button type="button" variant="outline" className="min-h-11" aria-label="Move mask right" onClick={() => commitSelectedMask({ ...selectedMask, ...updateRect(selectedMask, { x: selectedMask.x + 0.01 }) })}>→</Button>
                                    </div>
                                    <Button type="button" variant="destructive" className="min-h-11" onClick={() => { commit({ ...history.present, masks: history.present.masks.filter((mask) => mask.id !== selectedMask.id) }, "Mask deleted."); setSelectedMaskID(null); }}>Delete selected mask</Button>
                                </div>
                            ) : <p className="mt-3 rounded-xl bg-muted px-3 py-3 text-sm text-muted-foreground">Add or select a mask to adjust it without dragging.</p>}
                        </div>
                    </div>
                </aside>
            </div>

            <div className="mt-5 flex flex-wrap gap-2 border-t border-border pt-4">
                <Button type="button" variant="ghost" className="min-h-11" disabled={phase === "exporting"} onClick={cancel}>Cancel</Button>
                <Button type="button" variant="link" className="min-h-11" disabled={phase === "exporting"} onClick={manualEntry}>Manual entry</Button>
            </div>
            <div className="receipt-workspace-actions mt-4 flex gap-2 sm:justify-end">
                {phase === "exporting" ? <Button type="button" variant="outline" className="min-h-12" onClick={cancelExport}>Cancel preparation</Button> : null}
                <Button type="button" className="min-h-12 w-full text-base sm:w-auto" disabled={phase === "exporting"} onClick={() => void exportReceipt()}>
                    {phase === "exporting" ? <><Spinner aria-hidden="true" /> Preparing…</> : "Scan receipt"}
                </Button>
            </div>
            <p className="mt-3 text-sm text-muted-foreground" role="status" aria-live="polite">{status}</p>
        </section>
    );
}
