# Features

So far we have services and collaborations. We know who talks to who. But if someone asks "what does this system do?", all we can show them is a list of calls between services. That tells you nothing about the business value.

A feature is a business capability — the reason these services talk to each other in the first place. In our case, everything we've documented so far exists because of one thing: **ordering coffee**.

```
feature ordering: "Process coffee orders from placement to delivery" {
    collaboration orders -> beans {
        description: "Validate bean availability for the order"
    }
    collaboration orders -> barista {
        description: "Send accepted order for brewing"
    }
    collaboration barista -> beans {
        description: "Fetch beans for brewing"
        cardinality: one to many by bean_type
    }
    collaboration barista -> orders {
        description: "Report coffee is delivered"
    }
}
```

Same collaborations as before, but now they're grouped under a feature. Every collaboration inside the block automatically belongs to `ordering`. You can trace this feature across the entire system and see every service involved.

Check [`architecture/orgs/coffeeshop/services.arch`](architecture/orgs/coffeeshop/services.arch) for the full file.

```bash
archlang generate ./architecture --out ./generated --package generated
```

Now we know *what* the system does and *which services* are involved. But the ordering feature has phases — you validate first, then brew, then deliver. Right now the collaborations are just a flat list. That's where flows come in.
