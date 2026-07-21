package main

const (
	orderEvaluateHookID = "sforum.commerce-workflow.hook.order-evaluate"
	orderCreatedHookID  = "sforum.commerce-workflow.hook.order-created"
	orderEvaluateName   = "sforum.commerce-workflow.order.evaluate"
	orderCreatedName    = "sforum.commerce-workflow.order.created"
	orderEvaluateInput  = "sforum.commerce-workflow.hook.order-evaluate.input@1"
	orderEvaluateResult = "sforum.commerce-workflow.hook.order-evaluate.result@1"
	orderEvaluatePatch  = "sforum.commerce-workflow.hook.order-evaluate.result.patch@1"
	orderCreatedInput   = "sforum.commerce-workflow.hook.order-created.input@1"

	serviceID       = "sforum.commerce-workflow.service.orders"
	serviceVersion  = "1.0.0"
	serviceRequest  = "sforum.commerce-workflow.service.orders.request@1"
	serviceResponse = "sforum.commerce-workflow.service.orders.response@1"

	jobID      = "sforum.commerce-workflow.job.settle"
	jobKind    = "sforum.commerce-workflow.settle"
	jobHandler = "sforum.commerce-workflow.job.settle"
	jobPayload = "sforum.commerce-workflow.job.settle.payload@1"

	commandID           = "sforum.commerce-workflow.command.settle-once"
	commandHandler      = "sforum.commerce-workflow.command.settle-once"
	commandInputSchema  = "sforum.commerce-workflow.command.settle-once.input@1"
	commandResultSchema = "sforum.commerce-workflow.command.settle-once.result@1"

	guardOwnerID = "sforum.commerce-workflow.guard.owner"

	routeOrdersID         = "sforum.commerce-workflow.route.orders"
	routeManagedOrdersID  = "sforum.commerce-workflow.route.managed-orders"
	routeTopicsBeforeID   = "sforum.commerce-workflow.route.topics-before"
	routeTopicsAfterID    = "sforum.commerce-workflow.route.topics-after"
	routeTopicsFilterID   = "sforum.commerce-workflow.route.topics-filter"
	routeCreateWrapID     = "sforum.commerce-workflow.route.create-topic-wrap"
	routeCreateReplaceID  = "sforum.commerce-workflow.route.create-topic-replace"
	routeEventsID         = "sforum.commerce-workflow.route.events"
	routeStreamID         = "sforum.commerce-workflow.route.stream"

	ordersResponseSchema        = "sforum.commerce-workflow.route.orders.response@1"
	managedOrdersResponseSchema = "sforum.commerce-workflow.route.managed-orders.response@1"
)
