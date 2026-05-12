# Cardinality

Not every collaboration is one-to-one. When the barista fetches beans, it might need beans from multiple suppliers. When orders notifies about a state change, it might notify multiple channels.

ArchLang lets you express this:

```
collaboration barista -> beans {
    description: "Fetch beans for brewing"
    cardinality: one to many by bean_type
}
```

This says: for each order, the barista may call beans multiple times — once per `bean_type`. The `by` clause tells you what drives the fan-out.

Cardinality options:
- `one to one` — the default, one call per interaction
- `one to many by <key>` — one source triggers multiple calls, partitioned by a key

Check [`architecture/orgs/coffeeshop/services.arch`](architecture/orgs/coffeeshop/services.arch) for the full file.

```bash
archlang generate ./architecture --out ./generated --package generated
```

Our collaborations now carry more detail. But we still don't know *why* these services talk to each other — what business capability they serve. That's what features are for.
