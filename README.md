
<div align="center">

# Gundo

### A Modular SaaS-Based Command-and-Control Framework

An open-source C2 framework built in Go for authorized red teaming and adversary emulation, using third-party SaaS platforms to communicate with remote agents.

<sub>Go &nbsp;·&nbsp; GitHub Issues &nbsp;·&nbsp; Modular C2</sub>

<br/>

[**Getting Started**](#getting-started) &nbsp;·&nbsp;
[**Documentation**](#documentation) &nbsp;·&nbsp;
[**Architecture**](#architecture) &nbsp;·&nbsp;
[**Contributing**](#contributing)

</div>

<br/>

---

## Overview

Gundo is an open-source command-and-control (C2) framework built in Go for authorized red teaming and adversary emulation. It enables operators to manage remote agents, execute commands, and retrieve results using third-party Software-as-a-Service (SaaS) platforms as communication channels.

### Why Gundo?

Traditional C2 frameworks often rely on agents communicating directly with dedicated command servers. These connections can expose C2 infrastructure and provide defenders with identifiable network activity to investigate or block.

Gundo takes a different approach by routing C2 communication through legitimate SaaS platforms, beginning with GitHub Issues.

Instead of connecting directly to a C2 server, agents communicate with GitHub's API, allowing C2 activity to blend with ordinary SaaS traffic.

This provides red team operators with an alternative communication channel for authorized engagements while maintaining the ability to issue commands, manage agents, and retrieve results like a traditional C2.

---

## Features

- **Remote Task Execution** — Issue commands to remote agents and retrieve execution results.
- **Agent Management** — Manage remote agents and monitor their activity through the operator interface.
- **SaaS-Based Communication** — Exchange tasks and results through GitHub Issues instead of direct agent-to-controller connections.
- **Asynchronous Operations** — Agents periodically check for new tasks and return results without maintaining persistent connections.
- **Modular Transports** — Support for independently developed SaaS communication methods, with GitHub Issues as the initial implementation.
- **Operator Interface** — Manage agents, submit tasks, and review execution results from a centralized interface.

---

## Architecture

<div align="center">

<!-- Export your diagram from draw.io as an SVG and place it at:
     assets/architecture/gundo-overview.svg

     Then replace this placeholder with:
     <img src="assets/architecture/gundo-overview.svg" alt="Gundo High-Level Architecture" width="850">
-->

*High-level architecture diagram coming soon.*

</div>

Gundo consists of three main components:

| Component | Responsibility |
|:---|:---|
| **Controller** | Manages operator requests, agent communication, and execution results. |
| **Transport** | Exchanges commands and results through a supported SaaS API. |
| **Agent** | Retrieves commands, executes tasks, and returns results. |

### Communication Flow

1. The operator submits a command through Gundo's interface.
2. The controller publishes the command through the configured SaaS transport.
3. The agent retrieves the command during its next check.
4. The agent executes the task and publishes the result.
5. The controller retrieves the result and presents it to the operator.

The controller and agent exchange information asynchronously through the SaaS platform without maintaining a direct connection.

---

## Getting Started

Gundo is intended for authorized red team engagements, adversary emulation, and controlled testing environments.

A complete setup guide will cover installation, configuration, deployment, and running your first task.

**Getting Started guide coming soon.**

---

## Documentation

Whether you're new to C2 frameworks or interested in the underlying implementation, Gundo's documentation will provide a path from introductory concepts to detailed engineering references.

| Guide | Description |
|:---|:---|
| **Getting Started** | Install and configure Gundo, deploy an agent, and execute your first task. |
| **Core Concepts** | Understand command-and-control fundamentals, SaaS transports, and asynchronous communication. |
| **Architecture** | Explore the controller, agent, transport layer, and system communication flow. |
| **Engineering** | Understand implementation details, communication protocols, reliability, and design decisions. |
| **Developer Reference** | Explore Go packages, interfaces, functions, configuration, and code documentation. |

*A dedicated documentation website is being developed.*

---

## Contributing

Gundo is an open-source framework welcoming contributions to its core functionality, transport integrations, testing, documentation, and developer experience.

Contribution guidelines and development setup instructions will be provided in `CONTRIBUTING.md`.

Related defensive research and detection engineering efforts are maintained in separate repositories.

---

## Responsible Use

Gundo is intended exclusively for authorized red teaming, adversary emulation, educational environments, and controlled security assessments.

Users are responsible for obtaining proper authorization before deployment and complying with applicable laws and third-party service policies.

---

## License

License information will be available in the repository's `LICENSE` file once selected.
