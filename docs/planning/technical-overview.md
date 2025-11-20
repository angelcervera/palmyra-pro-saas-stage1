# Palmyra Pro SaaS Platform – Technical Overview

## 1. Introduction & Purpose

This document provides a **high-level technical overview** of the Palmyra Pro SaaS Platform for technical and architecture teams who need to understand:

* What the platform does in terms of **traceability** and **data handling**.
* How external systems can **integrate** with it.
* How **security**, **multi-tenancy** and **on-chain anchoring** are approached at a conceptual level.

It focuses on the current capabilities of Palmyra Pro as a **generic, schema-driven traceability platform** (current stage of the product), rather than on future roadmap items.

### 1.1 Audience

The primary audience for this document is:

* **Software architects and senior engineers** evaluating integration with Palmyra Pro.
* **Technical decision-makers** who need to understand the platform’s role in a wider system landscape (existing ERP systems, data platforms, programme tools, etc.).

The document assumes familiarity with modern web APIs and multi-tenant SaaS platforms, but does not require detailed knowledge of specific technologies used internally.

### 1.2 Scope

This document covers:

* The **business and technical context** in which Palmyra Pro operates.
* A **high-level architecture** of the Palmyra Pro SaaS Platform, including its integration with the external Winter Protocol API and the Cardano blockchain for selected transactions.
* The **integration model** for external systems, based on schemas and configuration.
* The **security and multi-tenancy model**, including tenant isolation and access control.
* An overview of **operations and reliability** at a conceptual level.
* Examples of **headless commodity implementations** for honey and cocoa.

The goal is to provide enough information for an architecture team to:

* Understand how Palmyra Pro fits into an overall system landscape.
* Assess the implications for integration, security and data flows.
* Identify which areas would require deeper discussion in subsequent technical sessions.

### 1.3 Out of Scope

To protect internal intellectual property and keep the focus on integration and behaviour rather than implementation, this document **intentionally excludes IP-sensitive details**. In particular, it does **not** describe:

* Detailed internal **data models**, **database schemas** or storage layouts.
* Internal **service topology**, deployment diagrams or infrastructure specifics.
* Proprietary **algorithms**, optimisation strategies or other implementation details that are part of ZenGate’s intellectual property.
* Detailed configuration of individual programmes or commodities beyond the examples provided.
* Internal design or implementation details of **Winter Protocol** beyond what is required to understand how Palmyra Pro integrates with it.

Those topics can be addressed separately, and at a finer level of detail, under appropriate agreements if and when required.

---

## 2. Context & Use Cases

### 2.1 Business Context

Palmyra Pro is a multi-tenant SaaS platform focused on **traceability of data across supply chains and programmes**.
It is designed to collect, normalise and connect information from multiple organisations involved in a value chain (for example: producers, aggregators, processors, buyers, auditors or programme owners), and to maintain a consistent trace of how products and claims move and change over time.

In its current stage, Palmyra Pro operates as a **generic, schema-driven traceability engine**:

* It does not hard-code commodity-specific concepts such as “lot” or “shipment” into the platform itself.
* Instead, each programme or commodity is modelled through configurable schemas and relationships registered in the platform.
* Selected state transitions can be anchored on the Cardano blockchain (via the external Winter Protocol API) to provide tamper-evident guarantees, while full data remains off-chain inside the platform.

This allows different programmes and supply chains to share the same underlying platform while keeping their data, configuration and workflows independent.

### 2.2 Typical Actors

Although the platform is generic, the same broad categories of actors appear in most deployments:

* **Producers / Farmers / Primary Suppliers**
  Capture origin data (production, harvesting, farm-level activities) and provide evidence related to practices, volumes and deliveries.

* **Aggregators / Processors / Exporters**
  Combine, transform or move product between locations, updating traceability records to reflect changes in ownership, location, volume or composition.

* **Importers / Buyers / Brand Owners**
  Consume traceability information to support procurement decisions, compliance reporting, risk assessment and transparency towards customers or regulators.

* **Programme Owners / Certification Bodies**
  Define requirements, indicators and data structures for a programme, review traceability records, and monitor compliance at scale.

* **Technical Integrators**
  Implement and maintain system-to-system integrations between local **ERP systems**, farm management tools, mobile apps, analytical systems and the Palmyra Pro SaaS Platform.

These actors may interact through their own systems (ERP systems, mobile apps, portals) or through user interfaces built specifically on top of Palmyra Pro.

### 2.3 Example Use Case: End-to-End Traceability Across a Supply Chain

A typical use case for Palmyra Pro is **end-to-end traceability** across a multi-step supply chain:

1. **Data capture at origin**
   Producers or their local systems submit records describing production, harvesting or initial consolidation. These records conform to schemas defined for the programme (for example: origin, quantities, basic quality parameters).

2. **Updates during aggregation and processing**
   As product moves through aggregators, processors or exporters, their systems submit additional records:

    * Linking new records to previous ones (e.g. “this shipment is composed of these inputs”).
    * Recording changes such as transformation, blending, splitting or re-packaging.

3. **Traceability graph in Palmyra Pro**
   Palmyra Pro maintains an internal traceability graph per tenant:

    * Connecting records according to the configured schemas and relationships.
    * Allowing queries that reconstruct the chain of activity from origin to later stages.

4. **Anchoring selected events on-chain**
   For specific milestones (for example, programme-defined control points or certified shipments), selected state transitions are mapped into Winter Protocol messages and anchored on the Cardano blockchain.
   The on-chain commitments act as tamper-evident checkpoints, while detailed business data remains in the platform.

5. **Downstream consumption**
   Buyers, programme owners or analytical tools query the Palmyra Pro SaaS Platform to:

    * Retrieve traceability paths for specific products or claims.
    * Check which key events have associated on-chain commitments.
    * Use the data for reporting, dashboards or downstream transparency.

### 2.4 Example Use Case: Programme Monitoring & Reporting

Another common use case is **programme-level monitoring** across many participants:

1. **Programme configuration**
   A programme owner defines:

    * The schemas that participants will use to submit data (e.g. production records, inspections, transactions).
    * The key relationships and indicators they care about (e.g. coverage, volumes, basic risk flags).

2. **Participant onboarding**
   Multiple organisations (tenants or sub-tenants, depending on the model) start submitting data according to the configured schemas, either via their existing systems or via dedicated UIs.

3. **Consolidated dataset in Palmyra Pro**
   Palmyra Pro stores all traceability records per tenant, while still allowing programme-level queries that operate across many participants (subject to access rules and data-sharing agreements).

4. **Reporting and analytics**
   Programme owners and authorised stakeholders can:

    * Query the Palmyra Pro SaaS Platform to obtain aggregated views (for example, total volume under a programme, basic flow patterns, coverage by region).
    * Export data into their own analytics or BI tools for further processing.

In both examples, the key idea is the same: **Palmyra Pro provides a generic, schema-driven traceability backbone**, which can be adapted to different programmes and supply chains without changing the underlying platform architecture.

---

## 3. High-Level Architecture (Current Stage)

### 3.1 Overview

The current Palmyra Pro platform is delivered as a multi-tenant SaaS.
It exposes the **Palmyra Pro SaaS Platform** for reading and writing traceability data, backed by a **persistence layer** and **tenant-isolated databases**.

Palmyra Pro is **integrated with Winter Protocol**, an external API owned by ZenGate. Palmyra uses Winter Protocol to anchor selected state transitions on the **Cardano blockchain**, providing transparent and tamper-evident traceability.

Winter Protocol itself is not part of the Palmyra Pro codebase; Palmyra acts as a client of this external service. The **schemas used by Winter Protocol are also registered within the Palmyra Pro SaaS Platform**, so that the persistence layer can validate data and maintain consistency between internal records and blockchain-anchored representations.

### 3.2 Main Components

* **Client Applications (Commodity UIs / Partner Tools)**
  External web or mobile applications used by producers, aggregators, buyers, certification bodies, or programme managers.
  These applications:

    * Capture operational data (records related to production, movements, inspections, transformations, etc.).
    * Visualise traceability information (supply chain paths, indicators, certificate status).
    * Communicate with Palmyra Pro exclusively via the **Palmyra Pro SaaS Platform** (its public APIs).

* **Palmyra Pro SaaS Platform**
  The public surface of Palmyra Pro that external systems integrate with. At a high level, it provides:

    * Endpoints for **submitting** schema-based records and state changes (e.g. creating or updating traceability records, attaching evidence).
    * Endpoints for **querying** traceability views, indicators and programme data.
    * Tenant/organisation-scoped management functions where required.
    * A formally described API (OpenAPI) that is shared during concrete integration work.

* **Persistence Layer**
  A logical service inside Palmyra Pro that:

    * Validates incoming payloads against registered schemas (including those derived from Winter Protocol).
    * Normalises and enriches data into Palmyra’s internal domain model.
    * Maintains full audit trails (who did what, when, and in which context).
    * Orchestrates the publishing of relevant state changes to the blockchain via Winter Protocol.

* **Tenant Databases**
  Each organisation (tenant) has its own logically isolated data store:

    * All traceability records, attachments and configuration are stored in a tenant-scoped database.
    * The persistence layer enforces tenant boundaries; cross-tenant data access is not permitted.
    * This model supports data residency and per-tenant lifecycle management (onboarding, archival, deletion).

* **Winter Protocol Integration & Blockchain Anchoring**
  Palmyra Pro integrates with the external **Winter Protocol API** to:

    * Construct protocol-compliant messages representing key state transitions or attestations.
    * Submit those messages for anchoring on the **Cardano blockchain**, creating immutable, verifiable records.
    * Store the identifiers and references needed to prove that specific supply-chain states were anchored at a given time.

### 3.3 Logical Data Flow

At a high level, the current data flow is:

1. A client application calls the **Palmyra Pro SaaS Platform** to submit or update traceability data (for example, create a new record representing a production event or shipment).
2. The **persistence layer**:

    * Validates the payload against the relevant schemas.
    * Applies programme rules and updates the tenant’s data in the **tenant database**.
3. For selected events or milestones, the persistence layer:

    * Builds a Winter-compliant message.
    * Sends it to the **Winter Protocol API**, which in turn anchors the corresponding commitment on the **Cardano blockchain**.
4. Aggregated views, indicators and reports are computed from the tenant data and exposed through:

    * The **Palmyra Pro SaaS Platform**, for consumption by client applications or other partner systems.

This separation keeps integrations simple (external systems talk only to the Palmyra Pro SaaS Platform), while leveraging Winter Protocol and Cardano as an independent trust layer.

---

## 4. Integration Model (Stage 2 – Generic, Schema-Driven)

This section focuses on how external systems interact with Palmyra Pro.
The key point: **partners integrate with the Palmyra Pro SaaS Platform; Palmyra Pro maintains internal traceability for all transactions and handles the interaction with Winter Protocol and the blockchain for selected transactions only.**

### 4.1 External Actors

Typical external systems that connect to Palmyra Pro include:

* Producer/exporter systems (ERP systems, farm management tools, mobile or offline apps).
* Programme and certification tools (audit management, compliance systems, dashboards).
* Buyer/importer systems (procurement, inventory, ESG reporting, customer-facing transparency portals).

All these systems communicate with Palmyra Pro through the **Palmyra Pro SaaS Platform**.

### 4.2 Integration with the Palmyra Pro SaaS Platform (Stage 2 – Generic, Schema-Driven)

In the current stage, Palmyra Pro provides a **generic, schema-driven API** rather than hard-coded concepts in the platform itself.

* **Schema-driven payloads**

    * Each programme or commodity defines its own data structures (records, events, relationships, attachments) using schemas registered in the Palmyra Pro SaaS Platform.
    * These schemas can be aligned with, or derived from, Winter Protocol schemas where blockchain anchoring is required.

* **Generic traceability operations**
  Through the Platform, external systems can:

    * Submit **records** and **state changes** that conform to the configured schemas.
    * Link records together to represent **chains of activity** (for example, from origin to final destination) without the platform enforcing a specific commodity model.
    * Attach supporting artefacts (documents, metadata) in a generic way.

* **Single public surface**

    * All of this is exposed through a single API surface (the Palmyra Pro SaaS Platform), described using OpenAPI.
    * The same generic operations and endpoints are reused across commodities and programmes; differences are expressed via schemas and configuration, not separate APIs.

* **Usage patterns**

    * Browser-based UIs and mobile apps send and retrieve schema-based records through the Platform.
    * Backend systems integrate server-to-server, pushing traceability data and querying it using the same schema-driven model.

This design allows Palmyra Pro to remain **commodity-agnostic in Stage 2**, while still supporting very different data models and workflows through configuration.

### 4.3 Palmyra Pro ↔ Winter Protocol ↔ Blockchain

The interaction with Winter Protocol and the Cardano blockchain for **selected transactions** is **handled entirely within Palmyra Pro**, while **traceability for all transactions is managed internally**:

* Palmyra Pro keeps a complete internal traceability record of all schema-based records and state changes in its tenant databases, regardless of whether they are anchored on-chain.
* For selected milestones or critical events, Palmyra Pro maps internal records into Winter Protocol messages using the registered schemas in the SaaS Platform.
* These messages are sent to the **Winter Protocol API**, which anchors commitments on the **Cardano blockchain**.
* Palmyra Pro stores the resulting identifiers and references so that:

    * External systems can query which records have been anchored.
    * Independent verification is possible without exposing internal implementation details or raw data on-chain.

Partner systems do not need to integrate directly with Winter Protocol to benefit from this; they simply consume traceability information and attestation status via the **Palmyra Pro SaaS Platform**.

### 4.4 Typical End-to-End Scenario

1. A producer’s system or a partner-provided UI submits a new traceability record (for example, a production or shipment record) to the **Palmyra Pro SaaS Platform**, using the schemas configured for that programme.
2. Palmyra Pro validates the payload against the relevant schema, then persists it in the appropriate **tenant database**, where it becomes part of the full internal traceability graph.
3. When that record reaches a configured key milestone (for example, a certified shipment or aggregated lot) that should be anchored on-chain, Palmyra Pro:

    * Generates a Winter Protocol message representing that state.
    * Submits it through the **Winter Protocol API** to be anchored on **Cardano**.
4. A partner system later calls the **Palmyra Pro SaaS Platform** to:

    * Retrieve the traceability records and their relationships as stored internally.
    * Check which key records have been anchored on the blockchain and obtain the corresponding references.

---

## 5. Security & Multi-Tenancy

Palmyra Pro is designed as a multi-tenant SaaS platform where multiple organisations (tenants) share the same platform while keeping their data, configuration, and access strictly isolated.
This section describes the **logical security model** and **multi-tenancy approach** at a high level. It intentionally does not expose detailed implementation or infrastructure specifics.

### 5.1 Tenant Isolation

* **Tenant-scoped data stores**
  Each organisation is assigned its own logically isolated data space. All traceability records, attachments and configuration for a tenant are stored and processed within that tenant scope.

* **Isolation enforced in the platform layer**
  The Palmyra Pro SaaS Platform enforces tenant boundaries at the application and persistence layer:

    * Every request is associated with exactly one tenant context.
    * All read/write operations are evaluated within that context.
    * Cross-tenant access to data is not allowed.

* **Lifecycle management**
  Onboarding, suspension, archival and deletion of tenant data are handled per organisation. This enables clear separation of responsibilities and supports programme- or region-specific data residency requirements where applicable.

### 5.2 Authentication & Authorisation

* **Authenticated access to the Palmyra Pro SaaS Platform**
  All access to the Palmyra Pro SaaS Platform itself is authenticated. Client applications and backend systems must present valid credentials to read or write tenant data through the Platform APIs.

* **Public access to on-chain commitments (by design)**
  When selected events are anchored on the Cardano blockchain via Winter Protocol, the resulting on-chain commitments are **public**, as is typical for a public ledger.

    * These commitments are designed to be minimal and do **not** expose raw traceability records.
    * Full business data remains inside Palmyra Pro’s tenant databases and is only accessible through authenticated Platform access.

* **Tenant-bound credentials**
  API keys, tokens or other credentials are always bound to a specific tenant. Even if a credential is compromised, it cannot be used to access other tenants’ data.

* **Role-based access control (RBAC)**
  Within each tenant, access is controlled using roles and permissions:

    * User and system identities are associated with roles (for example, producer, auditor, programme manager, system integration).
    * Roles determine which operations and datasets are accessible within that tenant.
    * This allows a tenant to separate duties internally (for example, between data entry and approval).

### 5.3 Data Protection

* **Separation of data and control plane**
  Traceability records and configuration are stored in the tenant data layer, while authentication, authorisation and routing are handled separately in the platform layer. This helps reduce the risk of accidental data exposure across tenants.

* **Encryption in transit**
  Communication between external systems and the Palmyra Pro SaaS Platform is protected using industry-standard transport security (e.g. HTTPS/TLS). All API calls and integration endpoints require secure transport.

* **Encryption at rest (where applicable)**
  Sensitive data stored by the platform may be encrypted at rest, depending on the underlying infrastructure and hosting environment. This reduces the risk of exposure in the event of low-level storage compromise.

* **Minimal exposure of on-chain data**
  When selected events are anchored on the Cardano blockchain via Winter Protocol:

    * Only the minimal required commitments or references are written on-chain.
    * Raw business data is kept within Palmyra Pro’s tenant databases.
    * External verification is possible using references, without exposing full internal records.

### 5.4 Auditability & Traceability of Actions

* **Full history of changes**
  Palmyra Pro maintains a trace of changes for traceability records:

    * Who performed an operation.
    * When it was performed.
    * What record or state was affected.

* **Action-level audit trail**
  Administrative actions (for example, configuration changes, key programme operations) can be recorded in audit logs, allowing tenants and programme owners to review key activities over time.

* **Linking internal and on-chain states**
  For events that are anchored on the blockchain, Palmyra Pro stores:

    * Internal identifiers and state.
    * The corresponding Winter Protocol message reference and blockchain commitment.
      This makes it possible to reconstruct, for selected records, both the internal history and the associated on-chain evidence.

### 5.5 Shared Responsibility

Security and data protection are treated as a **shared responsibility** between the platform and the organisations using it:

* **Palmyra Pro**:

    * Provides tenant isolation, access control mechanisms, secure integration endpoints and audit capabilities.
    * Ensures that the interaction with Winter Protocol and the blockchain follows a minimal-disclosure principle.

* **Tenants and integration partners**:

    * Manage their user accounts, roles and credentials.
    * Ensure that their client applications and backend systems interact with the platform in a secure and compliant way (for example, protecting API keys and enforcing their own internal access policies).

This model allows multiple programmes and organisations to operate on the same platform, with a clear separation of data and responsibilities, while still benefiting from shared infrastructure and a consistent security model.

---

## 6. Operations & Reliability

This section describes how Palmyra Pro is operated as a SaaS platform at a conceptual level. It focuses on **operational practices and reliability characteristics**, not on specific infrastructure components or internal tooling.

### 6.1 Deployment Model

* **Managed SaaS platform**
  Palmyra Pro is operated as a managed, multi-tenant SaaS platform. Organisations consume the platform via the **Palmyra Pro SaaS Platform** (public API surface and associated services), without needing to run or manage the core platform themselves.

* **Environment separation**
  The platform is operated across separate environments (for example, test / staging / production), allowing:

    * Integration testing to be performed without impacting production workloads.
    * Progressive rollout of changes and configuration.

* **Configuration-driven behaviour**
  Programmes and commodities are primarily configured through **schemas and settings**, rather than bespoke code deployments. This reduces the operational risk when adapting the platform to new use cases.

### 6.2 Monitoring & Observability

* **Platform health monitoring**
  Core components of the Palmyra Pro SaaS Platform are monitored for availability and correct operation. The objective is to detect incidents, degraded behaviour or integration problems and respond appropriately.

* **Logging and diagnostics**
  Application logs are collected and used to:

    * Diagnose integration issues and unexpected behaviour.
    * Support analysis of failures.
    * Provide additional context for audit and compliance where applicable.

* **Traceability of requests**
  Requests can be correlated across components using identifiers, enabling investigation of specific flows (for example, from an incoming API call through to on-chain anchoring of a selected event) without exposing underlying implementation details.

### 6.3 Backups & Disaster Recovery

* **Data durability**
  Tenant data (traceability records, configuration and associated metadata) is stored on durable storage with regular backups. The goal is to minimise the risk of data loss due to operational incidents.

* **Backup and restore**
  Backups are taken on a scheduled basis. In the event of a failure affecting stored data, platform operators can restore from backups to a known-good state, subject to recovery objectives defined at the operational level.

* **Disaster recovery posture**
  The platform is designed so that, in the event of a major incident impacting a primary environment, core services and tenant data can be recovered according to defined recovery procedures. Exact recovery point and recovery time objectives can be discussed separately if required.

* **On-chain anchoring as an additional evidence layer**
  For selected transactions anchored via Winter Protocol on the Cardano blockchain, the on-chain commitments provide an additional layer of evidence that is independent of the platform’s own storage. This does not replace backups, but complements them for specific, critical events.

### 6.4 Change Management & Upgrades

* **Controlled rollout of changes**
  Changes to the Palmyra Pro SaaS Platform (for example, new features, schema extensions, performance improvements) follow a controlled rollout process, typically passing through non-production environments before reaching production.

* **Backward-compatible evolution where possible**
  Where feasible, changes to schemas and APIs are introduced in a backward-compatible way to:

    * Avoid breaking existing integrations.
    * Allow integrators time to adopt new capabilities.

* **Configuration-first evolution**
  Many programme- and commodity-specific adaptations are implemented through configuration and schemas rather than code changes. This helps:

    * Reduce deployment risk.
    * Keep the core platform stable while still allowing flexibility for different use cases.

* **Communication with integration partners**
  When changes may affect integrations (for example, new required fields in schemas, or deprecation of legacy constructs), they are communicated in advance to technical contacts so that they can assess and plan any necessary adjustments on their side.

---

## 7. Headless Commodity Implementations: Honey and Cocoa

While Palmyra Pro is designed and operated as a **generic, schema-driven traceability platform**, ZenGate also delivers **headless commodity solutions** built on top of the same core. These solutions provide programme- and commodity-specific configurations and user interfaces, while continuing to use the Palmyra Pro SaaS Platform as the single backend.

At the time of writing, two commodities are in scope:

* **Honey** – a headless implementation already running on the platform.
* **Cocoa** – a similar headless implementation under preparation, built on the same approach.

### 7.1 Headless Architecture Overview

In this context, *headless* means:

* The **core platform** (Palmyra Pro SaaS Platform, persistence layer, tenant databases, Winter Protocol integration) remains **generic and schema-driven**.
* For each commodity, ZenGate configures:

    * A set of **commodity-specific schemas** and relationships (for example, how production, aggregation, processing, quality checks and shipments are represented).
    * One or more **user interfaces and integration components** that are tailored to that commodity but still communicate exclusively via the Palmyra Pro SaaS Platform.
* There is **no separate backend per commodity**; instead, each commodity solution is a different configuration and UI layer on top of the same Palmyra Pro core.

This approach allows Palmyra Pro to support different supply chains without duplicating backend logic or fragmenting the platform.

### 7.2 Honey Supply Chain (Headless Implementation)

The honey solution is an example of a headless commodity implementation currently running on Palmyra Pro.

From a technical perspective:

* **Commodity-specific schemas**

    * Honey programmes define schemas for key records such as production at origin, aggregation at collection points, processing steps, shipments, and laboratory or quality results.
    * These schemas are registered in the Palmyra Pro SaaS Platform and used for validation, storage and traceability graph construction.

* **Actors and integrations**

    * Producers, aggregators, processors and buyers interact through:

        * Honey-specific web or mobile interfaces built on top of the Palmyra Pro APIs, and/or
        * Integrations between their existing ERP systems or operational tools and the Palmyra Pro SaaS Platform using the honey schemas.
    * All data is still persisted in the tenant databases managed by Palmyra Pro.

* **Headless behaviour**

    * The front-end applications and any middleware are **stateless clients** of the Palmyra Pro SaaS Platform.
    * No honey-specific logic is embedded **inside** the core platform; the differentiation is in schemas, configuration and UI, not in a separate backend.

* **On-chain anchoring for honey**

    * Selected honey-related events (for example, specific programme-defined milestones) can be anchored on the Cardano blockchain via Winter Protocol, using the same mechanisms described in earlier sections.
    * Internal records remain off-chain in Palmyra Pro; on-chain commitments are used as additional evidence points.

### 7.3 Cocoa Supply Chain (Headless Implementation in Preparation)

A similar **headless implementation for cocoa** is being prepared using the same architectural model:

* **Cocoa-specific schemas**

    * Cocoa programmes will define their own schemas for farm-level activities, fermentation, drying, grading, aggregation, shipping and other relevant steps.
    * These schemas will be configured in the Palmyra Pro SaaS Platform in the same way as for honey, allowing cocoa data to use the same generic traceability engine.

* **Reuse of core integration model**

    * Existing patterns described in the Integration Model (Section 4) apply directly: systems integrating for cocoa will use the same Palmyra Pro SaaS Platform, with cocoa-specific schemas and configuration.
    * User interfaces, if provided, will again be headless clients of the Palmyra Pro APIs rather than custom backends.

* **Optional on-chain anchoring**

    * As with honey, selected cocoa events can be anchored via Winter Protocol and Cardano, with minimal on-chain disclosure and full records retained in Palmyra Pro.

### 7.4 Benefits of the Headless Commodity Approach

Using Palmyra Pro as a **single generic platform** with **headless commodity implementations** on top provides several advantages:

* **Reuse of core capabilities**

    * Multi-tenancy, security, auditability, schema management and on-chain anchoring are implemented once and reused across commodities.

* **Commodity-specific flexibility without platform forks**

    * Honey and cocoa can each have their own data structures, workflows and UIs, without requiring separate backend systems or diverging codebases.

* **Clean integration options for partners**

    * Integration teams can:

        * Integrate directly with the Palmyra Pro SaaS Platform using commodity-specific schemas, or
        * Use the headless UIs and supplementary components provided for particular programmes.
    * In both cases, the technical contract remains the same Palmyra Pro Platform APIs.

* **Easier evolution over time**

    * As programmes or commodity requirements change, updates are handled primarily through schemas and configuration, while the underlying platform, security model and operational practices remain stable.

