
<div align="center">

# Gundo

### A Modular SaaS-Based Command-and-Control Framework

An open-source C2 framework built in Go that uses third-party SaaS platforms to communicate with remote agents without requiring a direct connection to the controller.

<sub>Built with Go · GitHub Issues Transport</sub>

<br/>

[**Get Started**](#getting-started) &nbsp;·&nbsp;
[**Documentation**](#documentation) &nbsp;·&nbsp;
[**Architecture**](#architecture) &nbsp;·&nbsp;
[**Contributing**](#contributing)

</div>

<br/>

---

## Why Gundo?

Traditional command-and-control (C2) frameworks commonly rely on agents communicating directly with operator-controlled servers.

**Gundo takes a different approach.**

Rather than requiring agents to connect directly to the controller, Gundo uses familiar Software-as-a-Service (SaaS) platforms, such as GitHub, to exchange commands and results.

The project explores how these everyday services can support C2 communication while introducing interesting engineering challenges around reliability, communication, and extensibility.

---

## Architecture

<div align="center">

<!-- Replace this placeholder with your draw.io diagram. -->
<!-- ![Gundo High-Level Architecture](assets/architecture/gundo-overview.svg) -->

*High-level architecture diagram coming soon.*

</div>

Gundo consists of three main components:

- **Controller** — Receives commands from the operator and manages communication with agents.
- **Transport** — Uses a SaaS platform to carry commands and results between the controller and agents.
- **Agent** — Retrieves commands, executes tasks, and sends the results back through the transport.

**How it works:** An operator submits a command, the controller publishes it through GitHub Issues, and the agent retrieves it during its next check. Once the task is completed, the result travels back through GitHub for the controller to collect.

This process is asynchronous, meaning the controller and agent don't need to maintain a continuous connection.

---

## What Gundo Does

**Send commands remotely**

Submit tasks through the controller and retrieve the results from remote agents.

**Communicate through GitHub**

Use GitHub Issues to exchange tasks and results without requiring agents to connect directly to the controller.

**Work without a continuous connection**

Agents periodically check for new tasks and publish results when they're ready. This is known as asynchronous communication.

**Built for additional SaaS transports**

Gundo separates how tasks are managed from how they're delivered, making it possible to add other SaaS communication methods.

---

## Getting Started

Gundo is designed for authorized cybersecurity research, educational environments, and controlled testing.

Installation, configuration, and a walkthrough of your first task will be available in the Getting Started documentation.

*Quickstart guide coming soon.*

---

## Documentation

Explore Gundo at your own pace, whether you're learning about C2 frameworks for the first time or interested in the underlying engineering.

| Guide | Description |
|:---|:---|
| **Getting Started** | Install Gundo, configure a transport, and run your first task. |
| **Core Concepts** | Learn how C2 works, what SaaS transports are, and why asynchronous communication matters. |
| **Architecture** | Understand the controller, agent, transport layer, and how they interact. |
| **Engineering** | Explore the design decisions, challenges, and implementation details behind Gundo. |
| **Developer Reference** | Find information about Go packages, interfaces, functions, and configuration. |

*A dedicated documentation website is in development.*

---

## Contributing

Gundo is an open-source project developed to explore security engineering, distributed systems, and modular software design.

Contributions to the framework, new transport integrations, testing, documentation, and developer experience are welcome.

Contribution guidelines will be available in `CONTRIBUTING.md`.

Related detection engineering and defensive research initiatives are maintained in separate repositories.

---

## Responsible Use

Gundo is intended for authorized security research, education, and controlled testing environments.

Users are responsible for obtaining permission before deploying or operating the framework and complying with applicable laws and third-party platform policies.

---

## License

License information will be available in the repository's `LICENSE` file once selected.
