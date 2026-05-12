# Flows and Steps

A feature tells you *what* the system does. A flow tells you *in what order*.

The ordering feature has a clear sequence: validate beans, brew the coffee, deliver the drink. Let's express that:

```
feature ordering: "Process coffee orders from placement to delivery" {
    flow order-lifecycle "The journey of a coffee order" {
        collaboration orders -> beans {
            description: "Validate bean availability for the order"
            step: validate
        }
        collaboration beans -> orders {
            description: "Confirm beans are validated"
            step: validate
        }
        collaboration orders -> barista {
            description: "Send accepted order for brewing"
            step: brew
        }
        collaboration barista -> beans {
            description: "Fetch beans for brewing"
            cardinality: one to many by bean_type
            step: brew
        }
        collaboration barista -> orders {
            description: "Report coffee is delivered"
            step: deliver
        }
    }
}
```

A **flow** groups collaborations into a named sequence. A **step** labels a phase within that flow. Multiple collaborations can share the same step — during `validate`, both the request and the confirmation happen. During `brew`, the barista fetches beans and starts brewing.

The compiler infers the step order from their position in the flow. You don't need to number them — just write them in sequence.

Check [`architecture/orgs/coffeeshop/services.arch`](architecture/orgs/coffeeshop/services.arch) for the full file.

```bash
archlang generate ./architecture --out ./generated --package generated
```

## What We've Built

Starting from three service declarations, we progressively added:

- **Collaborations** — who talks to who
- **Cardinality** — one-to-many relationships
- **Features** — why they talk (business capabilities)
- **Flows and steps** — in what order

This is a solid model for service-to-service communication. But there's a problem.

We're describing the coffee shop as if services call each other directly — request/response. That's not how this system actually works. In reality, these services communicate through **events**. They publish facts about what happened, and other services react independently.

For example, when the barista starts brewing, both **beans** (to fetch beans) and **orders** (to update the order status) react — at the same time, independently. A collaboration can't express that. It models a direct call from A to B, not "A states a fact and B and C both react."

ArchLang doesn't support events yet. In the next chapter, we'll change that.
