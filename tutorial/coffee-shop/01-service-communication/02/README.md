# Collaborations

Now that our services exist, let's say how they talk to each other.

A collaboration is an edge between two services — "this one calls that one." The simplest form:

```
collaboration orders -> beans
```

That's it. Orders talks to beans. We don't know why yet, we don't know how — just that there's a connection.

We can add a description to give it some context:

```
collaboration orders -> beans {
    description: "Validate bean availability for the order"
}

collaboration orders -> barista {
    description: "Send accepted order for brewing"
}

collaboration barista -> beans {
    description: "Fetch beans for brewing"
}

collaboration barista -> orders {
    description: "Report coffee is delivered"
}
```

Check [`architecture/orgs/coffeeshop/services.arch`](architecture/orgs/coffeeshop/services.arch) for the full file. Compile it:

```bash
archlang generate ./architecture --out ./generated --package generated
```

If you reference a service that doesn't exist — say `collaboration orders -> inventory` — the compiler will catch it. Every reference is validated at build time.

We now have services and connections. But some connections aren't one-to-one.
