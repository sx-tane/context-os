<script lang="ts">
    export let text = "";

    type Token =
        | { kind: "text"; text: string }
        | { kind: "link"; text: string; href: string }
        | { kind: "code"; text: string }
        | { kind: "bold"; text: string };

    $: tokens = tokenizeInlineMarkdown(text);

    function tokenizeInlineMarkdown(value: string): Token[] {
        const pattern =
            /\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)|(https?:\/\/[^\s)]+)|`([^`]+)`|\*\*([^*]+)\*\*/g;
        const out: Token[] = [];
        let index = 0;
        for (const match of value.matchAll(pattern)) {
            const start = match.index ?? 0;
            if (start > index) {
                out.push({ kind: "text", text: value.slice(index, start) });
            }
            if (match[1] && match[2]) {
                out.push({
                    kind: "link",
                    text: match[1],
                    href: trimURL(match[2]),
                });
            } else if (match[3]) {
                const href = trimURL(match[3]);
                out.push({ kind: "link", text: href, href });
            } else if (match[4]) {
                out.push({ kind: "code", text: match[4] });
            } else if (match[5]) {
                out.push({ kind: "bold", text: match[5] });
            }
            index = start + match[0].length;
        }
        if (index < value.length) {
            out.push({ kind: "text", text: value.slice(index) });
        }
        return out.length ? out : [{ kind: "text", text: value }];
    }

    function trimURL(value: string) {
        return value.replace(/[.,;:!?]+$/g, "");
    }
</script>

{#each tokens as token}
    {#if token.kind === "link"}
        <a href={token.href} target="_blank" rel="noreferrer">{token.text}</a>
    {:else if token.kind === "code"}
        <code>{token.text}</code>
    {:else if token.kind === "bold"}
        <strong>{token.text}</strong>
    {:else}
        {token.text}
    {/if}
{/each}

<style>
    a {
        color: #1f5f8b;
        font-weight: 700;
        text-decoration: none;
        overflow-wrap: anywhere;
    }

    a:hover {
        color: #1c1b18;
        text-decoration: underline;
    }

    code {
        border: 1px solid #d7d2c8;
        background: #f8f6ef;
        padding: 1px 4px;
        font-size: 0.92em;
    }
</style>
