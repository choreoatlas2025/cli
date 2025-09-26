// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package templates

import "embed"

// InitFS embeds starter assets for `choreoatlas init`.
//
//go:embed init/* init/**/*
var InitFS embed.FS

const (
	// RootFlowSpecTemplate is the path to the starter `.flowspec.yaml`.
	RootFlowSpecTemplate = "init/root.flowspec.yaml"
	// FlowDirectorySpecTemplate is the FlowSpec stored under flows/ for reference.
	FlowDirectorySpecTemplate = "init/flows/order-fulfillment.flowspec.yaml"
	// OrderServiceTemplate is the starter ServiceSpec for order service.
	OrderServiceTemplate = "init/services/order-service.servicespec.yaml"
	// InventoryServiceTemplate is the starter ServiceSpec for inventory service.
	InventoryServiceTemplate = "init/services/inventory-service.servicespec.yaml"
	// ShippingServiceTemplate is the starter ServiceSpec for shipping service.
	ShippingServiceTemplate = "init/services/shipping-service.servicespec.yaml"
	// SuccessfulTraceTemplate is the starter trace file.
	SuccessfulTraceTemplate = "init/traces/successful-order.trace.json"
	// GithubWorkflowMinimalTemplate is the minimal GitHub Actions workflow.
	GithubWorkflowMinimalTemplate = "init/github-actions/minimal.yml"
	// GithubWorkflowComboTemplate is the combined GitHub Actions workflow.
	GithubWorkflowComboTemplate = "init/github-actions/combo.yml"
	// ExamplesDir is the root for the optional examples tree.
	ExamplesDir = "init/examples"
)
