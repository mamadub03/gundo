
<div align="center">

# Gundo

### A Modular SaaS-Based Command-and-Control Framework

An open-source C2 framework built in Go for authorized red teaming and adversary emulation, using third-party SaaS platforms to communicate with remote agents.

<br>

**Go** &nbsp;·&nbsp; **GitHub Issues** &nbsp;·&nbsp; **Modular C2**

<br>

[**Getting Started**](#getting-started) &nbsp;&nbsp;·&nbsp;&nbsp;
[**Documentation**](#documentation) &nbsp;&nbsp;·&nbsp;&nbsp;
[**Architecture**](#architecture) &nbsp;&nbsp;·&nbsp;&nbsp;
[**Contributing**](#contributing)

</div>

<br>

---

## Overview

Gundo is a command-and-control (C2) framework that enables red team operators to manage remote agents, issue commands, and retrieve execution results through third-party Software-as-a-Service (SaaS) platforms.

Rather than requiring agents to connect directly to dedicated C2 infrastructure, Gundo uses SaaS APIs as intermediaries for communication. Its initial transport uses **GitHub Issues** to exchange tasks and results between agents and the controller.

The framework separates its operational, transport, and execution components, allowing communication methods to be developed independently of the core system.

---

## Features

- **Remote Task Execution** — Issue commands to remote agents and retrieve execution results through the controller.
- **SaaS-Based Communication** — Exchange tasks and results through GitHub Issues without direct agent-to-controller connections.
- **Agent Management** — Manage remote agents and monitor their activity through the operator interface.
- **Asynchronous Operations** — Agents periodically check for new tasks and return results without maintaining persistent connections.
- **Modular Transport Architecture** — Separates communication mechanisms from core functionality to support additional SaaS transports.
- **Operator Interface** — Manage agents, submit tasks, and review results through a centralized interface.

---

## Architecture

<div align="center">

<!-- Replace this placeholder with your draw.io SVG diagram. -->
<!-- ![Gundo High-Level Architecture](assets/architecture/gundo-overview.svg) -->

*High-level architecture diagram coming soon.*

</div>

Gundo is organized around three primary components:

| Component | Responsibility |
|:---|:---|
| **Controller** | Manages operator requests, remote agents, and execution results. |
| **Transport** | Exchanges tasks and results through a supported SaaS platform. |
| **Agent** | Retrieves tasks, executes instructions, and returns results to the controller. |

### Communication Flow

1. An operator submits a task through the interface.
2. The controller publishes the task using the configured SaaS transport.
3. The agent retrieves the task during its next check.
4. The agent executes the task and publishes its result.
5. The controller retrieves the result and presents it to the operator.

This approach allows agents and controllers to exchange information asynchronously without maintaining direct connections.

---

## Getting Started

Gundo is designed for authorized red teaming, adversary emulation, and controlled security assessments.

Setup instructions, configuration guidance, and a walkthrough of your first agent session will be available in the Getting Started documentation.

**Quickstart documentation coming soon.**

---

## Documentation

<div align="center">

**Learn the framework. Understand the architecture. Build with Gundo.**

</div>

| Guide | Description |
|:---|:---|
| **Getting Started** | Installation, configuration, and first agent session. |
| **Core Concepts** | Understand Gundo's SaaS communication model and key terminology. |
| **Architecture** | Explore the controller, agent, transport layer, and system workflow. |
| **Engineering** | Technical implementation, communication protocols, reliability, and design decisions. |
| **Developer Reference** | Interfaces, packages, configuration, and source-code references. |

A dedicated documentation website is in development.

---

## Contributing

Gundo is an open-source framework welcoming contributions to its core functionality, transport integrations, testing, documentation, and developer experience.

For development guidelines and contribution instructions, see `CONTRIBUTING.md` once published.

Related defensive security and detection engineering efforts are maintained in separate repositories.

---

## Responsible Use

Gundo is intended exclusively for authorized red teaming, adversary emulation, security assessments, and educational environments.

Operators are responsible for obtaining appropriate authorization before deployment and complying with applicable laws and third-party platform policies.

---

## License

License information will be available in the repository's `LICENSE` file once selected.
