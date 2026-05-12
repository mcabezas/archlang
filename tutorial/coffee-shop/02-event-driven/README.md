# Chapter 2: Event-Driven

In chapter 1 we modeled the coffee shop as services calling each other directly. That works, but it's not how this system actually works.

These services don't call each other. They publish facts — things that happened, past tense — and other services react to those facts independently. When the barista starts brewing, both beans (to fetch stock) and orders (to update status) react at the same time. A direct collaboration can't express that.

So we need events.

If you want to understand the rationale behind this event-driven coffee shop design, watch this: https://www.youtube.com/watch?v=glfpfoPBVCI

## New Syntax

ArchLang now supports events as first-class citizens. Three new things:

- `event` keyword — declare an event, like you declare a service
- `<-` operator — subscribe to an event (or receive from a service)
- `execute` property — what action runs in response (only valid on event collaborations)

Arrow direction tells you the relationship:
- `collaboration service -> Event` — service **publishes** the event
- `collaboration service <- Event` — service **subscribes** to the event

Events are nodes in the graph, just like services. Collaborations connect them. All existing metadata — features, flows, steps, descriptions — works with events.

## The Coffee Shop

Take a look at [`architecture/orgs/coffeeshop/services.arch`](architecture/orgs/coffeeshop/services.arch) and [`architecture/orgs/coffeeshop/ordering.arch`](architecture/orgs/coffeeshop/ordering.arch).

The system has 10 events and 3 services. Four events have no subscribers — the compiler will warn about them. They're still valid facts, just nobody reacts to them yet.

## Compiler Warnings

Events with no subscribers trigger a warning, not an error. Publishing a fact that nobody cares about is not wrong — it's an extension point. In the next chapter we'll add a notification service that subscribes to some of these orphan events.
