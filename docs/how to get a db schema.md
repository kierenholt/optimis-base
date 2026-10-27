### **School MIS Architecture & Specification Summary**

**1\. Finding a Starting Point for a School MIS Schema**

> * **Standard Frameworks:** The **Ed-Fi Alliance Unifying Data Model (UDM)** and **CEDS Integrated Data Store (IDS)** provide enterprise-grade, normalized relational models for K–12 data.  
> * **Open-Source References:** OpenEduCat (Python/PostgreSQL) and **Gibbon Core** (PHP/MySQL) offer inspectable, production-tested database layouts.  
> * **Core Design Fundamentals:** A solid starting schema requires decoupling identity from roles (a shared persons or users table extended by student\_profiles and staff\_profiles), managing academic cycles temporally via academic\_year\_id, and isolating operational tables (scheduling, attendance, assessments).

**2\. Adapting Ed-Fi for the UK Market**

> * **Feasibility:** Ed-Fi can be extended using its proprietary **MetaEd** tool to generate UK-specific DDL, API endpoints, and descriptors.  
> * **Required Overhauls:**  
  * Identifiers: UPN, ULN, TRN, URN, and LA/Establishment codes.  
  * Descriptors: Replacing US federal race/ethnicity and attendance codes with DfE Common Basic Data Set (CBDS) values (e.g., statutory AM/PM roll-call marks instead of US period/minute attendance).  
  * Structural Shifts: Mapping US Districts to Multi-Academy Trusts (MATs) or Local Authorities (LAs), and standardized GPAs to Key Stages and GCSE/A-Level grades.  
> * **Verdict:** While technically possible, shoehorning Ed-Fi into UK requirements introduces heavy technical debt and complex ETL logic when exporting statutory returns.

**3\. The DfE CBDS Model vs. Open-Source Boilerplates**

> * **The DfE Reality:** The UK Department for Education does **not** provide a relational database schema (no SQL, DDL, or ER diagrams). It only publishes a data dictionary (**CBDS**) and XML schemas (.xsd) for the School Census and Common Transfer Files (CTF).  
> * **Why CBDS Is Still Better than Ed-Fi for the UK:** Direct alignment with CBDS column definitions and lookup codes makes statutory termly Census and CTF XML generation straightforward rather than an ETL nightmare.  
> * **The Role of Gibbon:** Because CBDS specifies *what* data to record but omits operational tables (timetabling, room assignments, period grids, pastoral notes, parent custody flags), **Gibbon Core** (gibbonedu/core on GitHub) serves as a British/international school operational blueprint. Its single gibbon.sql file provides battle-tested relational structures for day-to-day school workflows.

**4\. Architectural Comparison: Gibbon Core vs. OpenMIS UK Specification**

> * **Runtime & Type Safety:** Gibbon relies on a legacy, partially procedural PHP/MySQL stack, whereas OpenMIS UK uses **TypeScript on Node.js (Fastify)** with **Drizzle ORM** and **Zod** for end-to-end type safety from the database to the React 19 frontend.  
> * **Database & Isolation:** Gibbon uses shared tables on MySQL; OpenMIS UK implements **PostgreSQL 16+** with a **dedicated physical database per school**. This design physically guarantees no cross-tenant data leakage in MAT environments and eliminates the need for complex Row-Level Security (RLS).  
> * **Statutory Integrations:** Gibbon has zero out-of-the-box DfE statutory reporting. OpenMIS UK embeds an **in-process TypeScript/WASM engine** (fast-xml-parser / libxml-wasm) for CTF and Census XML, paired with scheduled automated JSON attendance rollups directly to the DfE API.  
> * **Infrastructure Footprint:** Gibbon traditionally runs on stateful LAMP/VPS hosts. OpenMIS UK deploys lightweight, unprivileged **OCI containers** to UK sovereign cloud regions (AWS ECS Fargate / Azure Container Apps) with **Caddy** auto-TLS for local dev, **pg-boss** to avoid a Redis container, and presigned S3/R2 URLs to eliminate internal object storage proxies.  
> * **Identity & Integrations:** Rather than relying on local passwords and costly third-party data middleware (e.g., Wonde), OpenMIS UK uses native **Staff SSO via Entra ID / Google Workspace (OIDC)** and **direct zero-tollbooth OAuth 2.0 client credentials**.

**5\. Key Takeaways & Strategic Recommendation**

> 1. **Retain the OpenMIS UK Technical Architecture:** Build natively with Fastify, PostgreSQL, Drizzle, React 19, and the physical database-per-school pattern.  
> 2. **Align Directly with DfE CBDS:** Adopt official DfE field definitions, regex constraints, and lookup lists from the outset to streamline statutory reporting.  
> 3. **Use Gibbon as an Operational Cheat Sheet:** Do not fork or deploy Gibbon's legacy PHP stack; instead, inspect gibbon.sql solely as a structural reference for complex school operations (e.g., 2-week rolling timetables, split-period classes, and multi-guardian contact rules).