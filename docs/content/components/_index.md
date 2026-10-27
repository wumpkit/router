---
title: "Components & Modals"
weight: 30
---

# Components & Modals

Wumpkit Router supports parameterized routing for Discord interactive UI components. Instead of parsing custom IDs by hand with string splits, you define routes like `ticket:{id}` and retrieve parameters with `ctx.Param("id")`.

* [Buttons](buttons/): Button commands, parameterized routes, and updating button messages.
* [Select Menus](select-menus/): Dropdown menus, reading selections, and handling choices.
* [Modals](modals/): Modal popup forms, text inputs, and reading submission data.
