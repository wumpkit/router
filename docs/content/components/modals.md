---
title: "Modals"
weight: 3
---

# Modal Form Submissions

Modals are popup forms containing text inputs. When a user completes the form and clicks submit, Discord fires a modal interaction.

## Defining a Modal Command

Implement the `ModalCommand` interface by embedding `router.ModalCommandConfig` and defining `Run(ctx *router.ModalCommandContext) error`:

```go
package main

import (
	"fmt"

	"github.com/wumpkit-go/router"
)

type FeedbackModalCommand struct {
	router.ModalCommandConfig
}

func NewFeedbackModalCommand() *FeedbackModalCommand {
	return &FeedbackModalCommand{
		ModalCommandConfig: router.ModalCommandConfig{
			CustomID: "feedback_modal",
		},
	}
}

func (c *FeedbackModalCommand) Run(ctx *router.ModalCommandContext) error {
	// read the text input value by its custom id
	subject := ctx.InputValue("input_subject")
	details := ctx.InputValue("input_details")

	user := ctx.Author()

	return ctx.ReplyEphemeral(fmt.Sprintf(
		"Thank you %s! We received your feedback regarding '%s'.",
		user.Username,
		subject,
	))
}
```

Register it on your router:

```go
r.MustRegisterModalCommand(NewFeedbackModalCommand())
```

## Parameterized Modals

Modals can also use route parameters:

```go
type UserReportModalCommand struct {
	router.ModalCommandConfig
}

func NewUserReportModalCommand() *UserReportModalCommand {
	return &UserReportModalCommand{
		ModalCommandConfig: router.ModalCommandConfig{
			CustomID: "report_modal:{target_id}",
		},
	}
}

func (c *UserReportModalCommand) Run(ctx *router.ModalCommandContext) error {
	targetID := ctx.Param("target_id")
	reason := ctx.InputValue("report_reason")

	return ctx.ReplyEphemeral(fmt.Sprintf(
		"Report submitted for user <@%s>. Reason: %s",
		targetID,
		reason,
	))
}
```
