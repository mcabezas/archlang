# Declaring Services

The **orders** service owns order data. It tracks the lifecycle of an order — from placement to delivery. It doesn't make coffee or check inventory. It just knows the state of your order.

The **beans** service owns coffee bean inventory. It validates whether the beans required for an order are available and manages stock when beans are consumed.

The **barista** service is the worker. It brews coffee and reports progress — from the moment it starts to the moment the drink is delivered.

In ArchLang, declaring them is one line each:

```
service orders
service beans
service barista
```

That's [`architecture/orgs/coffeeshop/services.arch`](architecture/orgs/coffeeshop/services.arch). The organization is inferred from the folder structure — these services belong to `coffeeshop`.

Try compiling it:

```bash
archlang generate ./architecture --out ./generated --package generated
```

No errors means the architecture is valid. We have three services. They don't talk to each other yet.
