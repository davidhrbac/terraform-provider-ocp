# Architecture and Internals

This document describes the internal architecture and design principles
of the OCP Terraform provider.

## High-Level Architecture

The provider is implemented as a thin Terraform layer on top of the OCP
GraphQL API.

```
Terraform
   |
   v
Resource / Data Source
   |
   v
internal/client (GraphQL)
   |
   v
OCP API
```

The design intentionally avoids additional abstraction layers.

## Terraform Layer

Resources and data sources are responsible for:

- Defining Terraform schemas
- Mapping schema attributes to GraphQL inputs
- Translating API responses into Terraform state
- Producing user-facing diagnostics

Terraform concepts (Create / Read / Update / Delete) are not exposed
outside this layer.

## API Client Layer

The `internal/client` package provides a minimal GraphQL client.

Responsibilities:

- HTTP transport
- Authentication headers
- TLS configuration
- JSON encoding and decoding

The client:
- is unaware of Terraform
- does not produce Terraform diagnostics
- returns technical errors only

## GraphQL Queries and Mutations

All interactions with the backend are performed via GraphQL queries and
mutations defined as string constants.

Many mutations return **union payloads**. These are handled explicitly to:

- surface validation errors clearly
- avoid ambiguous success states
- prevent silent failures

When a mutation returns a full object payload, the provider uses it directly
to initialize or refresh Terraform state without issuing an additional read.

## Update Semantics

Some resources separate update operations into mutually exclusive change groups.

Example: `ocp_virtual_host`

- sizing changes (CPU / memory)
- tier changes

Applying multiple change groups in a single Terraform apply is rejected
to preserve predictable ordering and API behavior.

## Error Handling Strategy

Error handling follows a layered approach:

- The API client returns technical errors
- Resource and data source layers wrap errors into user-facing diagnostics
- User-facing error messages follow the form:

```
failed to <action>: <reason>
```

## Design Principles

- Explicit over implicit
- Minimal abstraction
- Clear separation of responsibilities
- Idiomatic Go and Terraform style

## Adding a New Resource

When adding a new resource, follow this checklist:

1. Define Terraform schema with `Description` for all attributes
2. Implement CRUD functions
3. Add GraphQL queries and mutations
4. Handle union payloads explicitly
5. Add GoDoc comments for exported symbols
6. Follow established error message conventions
