# Tutorial: The Event-Driven Coffee Shop

A step-by-step guide to documenting a system architecture with ArchLang, using the classic event-driven coffee shop domain.

## The Domain

A coffee shop where three services — **orders**, **beans**, and **barista** — collaborate to process customer orders, validate bean inventory, brew coffee, and deliver drinks.

## What You'll Learn

1. How to declare services and model direct communication between them

## Prerequisites

- ArchLang installed (`go install github.com/mcabezas/archlang/cmd/archlang@latest`)

## Chapters

1. [Service Communication](01-service-communication/) — Declare the services and model their interactions using collaborations
2. [Event-Driven](02-event-driven/) — Model the coffee shop with events, publish/subscribe, and actions
