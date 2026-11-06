
<div align="center">

# Gundo

### A Modular SaaS-Based Command-and-Control Framework

An open-source C2 framework built in Go, using modular SaaS transports for asynchronous communication between operators and remote agents.

<br/>

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![Transport](https://img.shields.io/badge/Transport-GitHub_Issues-1A2838?style=flat-square&logo=github&logoColor=white)
![Architecture](https://img.shields.io/badge/Architecture-Modular-1A2838?style=flat-square)
![Status](https://img.shields.io/badge/Status-Active_Development-1A2838?style=flat-square)

<br/>

[**Getting Started**](#getting-started) &nbsp;·&nbsp;
[**Documentation**](#documentation) &nbsp;·&nbsp;
[**Architecture**](#architecture) &nbsp;·&nbsp;
[**Contributing**](#contributing)

</div>

<br/>

---

## Why Gundo?

Traditional command-and-control (C2) frameworks commonly communicate directly with operator-controlled infrastructure.

**Gundo explores a different approach.**

Rather than requiring direct agent-to-controller communication, Gundo exchanges tasks and results through third-party Software-as-a-Service (SaaS) APIs, beginning with GitHub Issues.

Built around independent operational, transport, and execution layers, Gundo provides a modular foundation for exploring alternative C2 communication architectures and the engineering challenges of distributed, asynchronous systems.

---

## Architecture

<div align="center">

<!-- Replace this text with your draw.io SVG export. -->

<!-- ![Gundo Architecture](assets/architecture/gundo-overview.svg) -->

*High-level architecture diagram coming soon.*

</div>

Gundo consists of three primary components:

- **Controller** — Coordinates operator tasks, agent communication, and results.
- **Transport** — Exchanges messages through a supported SaaS platform.
- **Agent** — Retrieves tasks, processes instructions, and publishes results.

The controller and agent communicate asynchronously through the transport rather than maintaining a direct connection.

[**Explore the architecture →**](#documentation)

---

## Core Capabilities

| Capability | Description |
|:---|:---|
| **Modular Architecture** | Separates operational logic, transport communication, and agent execution. |
| **Asynchronous Communication** | Exchanges tasks and results without persistent agent-to-controller connections. |
| **GitHub Issues Transport** | Uses GitHub's API as the initial SaaS communication channel. |
| **Task & Result Management** | Supports the task execution and result retrieval lifecycle. |
| **API-Aware Communication** | Incorporates polling, pagination, and rate-limit handling for SaaS API communication. |
| **Deployment & Packaging** | Provides a deployable and packaged framework for controlled environments. |

---

## Getting Started

Gundo is intended for authorized security research, education, and controlled lab environments.

Installation, configuration, and first-task walkthroughs will be available in the Getting Started documentation.

*Quickstart documentation coming soon.*

---

## Documentation

<div align="center">

**Explore the framework, from foundational concepts to implementation details.**

</div>

| Guide | Description |
|:---|:---|
| **Getting Started** | Installation, configuration, and your first task. |
| **Core Concepts** | C2 fundamentals, SaaS transports, and asynchronous communication. |
| **Architecture** | Controller, agent, transport layers, and system data flow. |
| **Engineering** | API communication, polling, pagination, reliability, and design decisions. |
| **Developer Reference** | Go packages, interfaces, functions, and configuration. |

*A dedicated technical documentation website is in development.*

---

## Contributing

Gundo welcomes contributions across framework development, transport integrations, testing, documentation, and developer experience.

Development guidelines and contribution instructions will be available in `CONTRIBUTING.md`.

Related detection engineering and defensive research projects are maintained separately from the core framework.

---

## Responsible Use

Gundo is intended for authorized cybersecurity research, educational environments, and controlled testing.

Users are responsible for obtaining proper authorization and complying with applicable laws and third-party platform policies.

---

## License

See the repository's `LICENSE` file once the project license is published.
