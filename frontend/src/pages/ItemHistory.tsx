import { type FormEvent, useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { apiFetch, getResponseErrorMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field, FieldLabel, FieldDescription, FieldGroup } from "@/components/ui/field";
import MobilePageHeader from "@/components/MobilePageHeader";

interface Item { itemId: string; expenseId: string; description: string; merchant: string; date: string; currency: string; quantity: string | null; unitPrice: string | null; lineTotal: string }
interface Page { items: Item[]; nextCursor?: string }

export default function ItemHistory() {
    const { id } = useParams();
    return <ItemHistorySearch key={id} id={id} />;
}

function ItemHistorySearch({ id }: { id: string | undefined }) {
    const [query, setQuery] = useState("");
    const [searched, setSearched] = useState("");
    const [page, setPage] = useState<Page>({ items: [] });
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");
    const request = useRef<AbortController | null>(null);
    useEffect(() => { request.current?.abort(); setPage({ items: [] }); setSearched(""); return () => request.current?.abort(); }, [id]);
    const search = async (more = false) => {
        const term = more ? searched : query.trim();
        if (!id || !term) return;
        request.current?.abort();
        const controller = new AbortController(); request.current = controller;
        setBusy(true); setError("");
        if (!more) { setPage({ items: [] }); setSearched(term); }
        try {
            const params = new URLSearchParams({ q: term });
            if (more && page.nextCursor) params.set("cursor", page.nextCursor);
            const response = await apiFetch(`/group/${id}/item-history?${params}`, { signal: controller.signal });
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Could not load item history. Try again."));
            const result = await response.json() as Page;
            if (!controller.signal.aborted) setPage(current => ({ items: more ? [...current.items, ...result.items] : result.items, nextCursor: result.nextCursor }));
        } catch (reason) { if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : "Could not load item history."); }
        finally { if (!controller.signal.aborted) setBusy(false); }
    };
    const submit = (event: FormEvent) => { event.preventDefault(); void search(); };
    const currencies = [...new Set(page.items.map(item => item.currency))];
    return <main className="mx-auto flex w-full max-w-4xl flex-col gap-6 px-4 pb-24 pt-6">
        <MobilePageHeader title="Item price history" backTo={`/group/${id}`} />
        <h1 className="hidden text-2xl font-semibold md:block">Item price history</h1>
        <p className="text-muted-foreground">Find previously saved items in this group. Prices stay in their original currency; each purchase is shown separately.</p>
        <form onSubmit={submit}><FieldGroup><Field><FieldLabel htmlFor="item-history-query">Item description</FieldLabel><Input className="min-h-11" id="item-history-query" value={query} onChange={event => setQuery(event.target.value)} maxLength={200} placeholder="For example, milk" /><FieldDescription>Search the descriptions you confirmed or entered manually.</FieldDescription></Field><Button className="min-h-11" type="submit" disabled={busy || !query.trim()}>{busy ? "Searching…" : "Search history"}</Button></FieldGroup></form>
        {error ? <div role="alert" className="flex flex-col gap-2"><p className="text-destructive">{error}</p><Button className="min-h-11" type="button" variant="outline" disabled={busy} onClick={() => void search(page.items.length > 0)}>Try again</Button></div> : null}
        {!busy && searched && !page.items.length && !error ? <p role="status">No saved items match “{searched}”. Try another description.</p> : null}
        {currencies.map(currency => <section key={currency} aria-label={`${currency} purchases`} className="flex flex-col gap-3"><h2 className="text-lg font-semibold">{currency} purchases</h2><ul className="flex flex-col gap-3">{page.items.filter(item => item.currency === currency).map(item => <li key={item.itemId} className="rounded-xl border border-border p-4"><Link to={`/expense/${item.expenseId}`} className="font-medium underline underline-offset-4">{item.description}</Link><p className="mt-1 text-sm text-muted-foreground">{item.merchant || "Merchant not recorded"} · {item.date}</p><p className="mt-2 font-medium">Line total: {item.lineTotal} {item.currency}</p>{item.quantity !== null ? <p className="text-sm">Quantity: {item.quantity}</p> : null}{item.unitPrice !== null ? <p className="text-sm">Unit price: {item.unitPrice} {item.currency}</p> : null}</li>)}</ul></section>)}
        {page.nextCursor ? <Button className="min-h-11" type="button" variant="outline" disabled={busy} onClick={() => void search(true)}>{busy ? "Loading…" : "Load more purchases"}</Button> : null}
        <Link to={`/group/${id}`} className="underline">Back to group</Link>
    </main>;
}
