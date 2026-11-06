
# Gundo

**A Modular SaaS-Based Command-and-Control Framework**

An open-source command-and-control (C2) framework built in Go, exploring asynchronous communication through modular Software-as-a-Service (SaaS) transports.

Gundo separates its operational, transport, and execution layers, enabling remote agents and controllers to exchange tasks and results through third-party SaaS APIs rather than communicating directly.

**Go** · **GitHub Issues** · **Modular Architecture** · **Active Development**

[Overview](#overview) · [Architecture](#architecture) · [Design Principles](#design-principles) · [Documentation](#documentation) · [Contributing](#contributing)

---

## Overview

### Command-and-control through everyday services.

Traditional C2 frameworks commonly rely on agents communicating directly with operator-controlled infrastructure.

Gundo takes a different approach.

Instead of requiring a direct connection between an agent and its controller, Gundo uses third-party SaaS platforms as intermediaries for exchanging tasks and results.

Its initial transport uses **GitHub Issues**, demonstrating how an existing API can support asynchronous communication between distributed components.

### Why Gundo?

Gundo explores what happens when command-and-control communication is treated as a modular, asynchronous system rather than a direct connection between an agent and a server.

By using SaaS APIs as transport infrastructure, the project investigates the engineering challenges of building reliable communication on platforms that were never designed for C2 operations.

The goal is to provide an extensible framework for studying C2 architecture, distributed systems, API communication, and the tradeoffs introduced by alternative transport mechanisms.

---

## Architecture

Gundo is designed around three primary components:

- **Controller:** Manages operator tasks and coordinates communication through the selected transport.
- **Transport:** Handles communication between the controller and agents using a third-party SaaS API.
- **Agent:** Retrieves tasks, processes instructions, and publishes execution results.

### System Overview

<!--
Replace this placeholder after exporting your diagram from draw.io.

Suggested file location:
assets/architecture/gundo-overview.svg

Then uncomment the following line:

![Gundo High-Level Architecture](assets/architecture/gundo-overview.svg)
-->

*High-level architecture diagram coming soon.*

### How It Works

1. **Task Creation** — An operator submits a task to the controller.
2. **Task Publication** — The controller publishes the task through the configured SaaS transport.
3. **Task Retrieval** — An agent polls the transport for new tasks.
4. **Task Execution** — The agent processes the task and generates a result.
5. **Result Delivery** — The agent publishes its result through the transport for the controller to retrieve.

This architecture allows the controller and agent to exchange information without maintaining a direct connection.

---

## Design Principles

### Modular Architecture

Gundo separates operational logic, transport communication, and agent execution into distinct components.

This separation allows individual components to be developed and maintained independently.

### Asynchronous Communication

Tasks and results are exchanged through an intermediary platform rather than relying on persistent connections between the controller and agent.

### Extensible Transport Layer

Gundo is designed to support multiple SaaS communication mechanisms through a common transport abstraction.

GitHub Issues serves as the initial transport, with additional integrations planned as the framework develops.

### Reliability and Maintainability

Gundo emphasizes predictable task processing, API efficiency, error handling, and maintainable implementations.

These considerations become especially important when building distributed communication systems around third-party APIs.

---

## Project Status

> **Status: Active Development**
>
> Gundo is an experimental framework under active development. Its interfaces, architecture, and functionality may change as the project evolves.

| Component | Status |
|---|---|
| Core controller | In development |
| GitHub Issues transport | Initial implementation |
| Agent execution | In development |
| Pagination and polling improvements | In progress |
| Credential management | Planned improvements |
| Deployment and packaging | Planned |
| Additional SaaS transports | Future development |

For implementation details and upcoming milestones, see the project roadmap when published.

---

## Documentation

Gundo's documentation is organized to support readers with different levels of technical experience.

Whether you're new to command-and-control frameworks or interested in the underlying engineering, you can explore the project at your own pace.

| Guide | What You'll Learn |
|---|---|
| **Getting Started** | Install Gundo, configure a transport, and follow your first task. |
| **Core Concepts** | Understand C2 fundamentals, asynchronous communication, and SaaS-based transports. |
| **Architecture** | Explore the controller, agent, transport abstraction, and end-to-end task lifecycle. |
| **Engineering** | Understand polling, pagination, API limitations, reliability, and architectural tradeoffs. |
| **Developer Reference** | Explore interfaces, data structures, configuration, and source-code documentation. |

*A dedicated documentation website is planned. Documentation links will be added as guides become available.*

---

## Contributing

Gundo is developed as an open-source cybersecurity engineering project with an emphasis on modular development, technical learning, and collaboration.

Contributions involving framework development, transport integrations, testing, documentation, and developer experience are welcome.

Contribution guidelines and development instructions will be provided in `CONTRIBUTING.md`.

Related detection engineering and defensive research initiatives are maintained separately from the core framework.

---

## Responsible Use

Gundo is intended for authorized cybersecurity research, education, and controlled testing environments.

Users are responsible for obtaining appropriate authorization before deploying or operating the framework and for complying with applicable laws and third-party service policies.

---

## License

License information will be added once the project's open-source license has been selected.
