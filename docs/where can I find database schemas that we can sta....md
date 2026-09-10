Because UK schools must report data according to legally defined formats, you do not need to invent database schemas from scratch. The three primary sources for battle-tested educational data models include:

### 1. The Official DfE Data Dictionaries & Specifications

The Department for Education publishes exhaustive technical data models that define every required field, data type, and valid enumeration used in UK schools.

* **DfE Common Basic Data Set (CBDS):**
  + *What it is:* The master technical database specification maintained by the DfE. It explicitly itemizes every data entity a school handles (Pupils, Addresses, Identifiers, SEN Status, Attendance codes, Characteristics) along with exact data types, maximum character lengths, regex patterns, and business rules.
  + *Where to find it:* Search for **"DfE Common Basic Data Set"** on GOV.UK. You can download the complete schema spreadsheet, which maps directly to PostgreSQL column types and Zod regex strings.
* **DfE Common Transfer File (CTF) XML Schema (XSD):**
  + *What it is:* The official XSD definition used when students transfer between schools.
  + *Why it matters:* It defines the relational hierarchy between students, historical attendance sessions, assessments, and medical flags. You can run pure TypeScript generators (like quicktype or csonv) directly on the official CTF .xsd file to spit out TypeScript interfaces immediately.
* **School Census Technical Specifications:**
  + *What it is:* Published annually by the DfE for software suppliers. It details the exact schemas required for termly census returns (Autumn, Spring, Summer).

### 2. Open-Source Educational Core Schemas

* **Wonde / EdTech API Specifications:**
  + Although you are bypassing brokers like Wonde to avoid tollbooths, their developer documentation publishes public JSON schema models representing UK schools (e.g., how classes, rooms, timetables, periods, and attendance marks relate to each other). These schemas provide an ideal, modern blueprint for structuring your Drizzle tables.
* **SIF (System Interoperability Framework) UK Data Model:**
  + The UK SIF data standard defines an enterprise-grade relational schema tailored to UK schooling (pupil personal records, school curriculum, attendance, and workforce/staff).
  + *Where to find it:* Maintained by the **A4L Community (Access 4 Learning)** under their open UK SIF specification downloads.

### 3. Open-Source Higher Ed / K-12 Relational Repositories

* **PostgreSQL / Open Source Community Projects:**
  + Repositories such as **Gibbon** (open-source school platform) or public Drizzle/Prisma schema collections for education portals.
  + While Gibbon uses MySQL and isn't strictly built for UK statutory reporting, its normalized relational structure (students, guardians, timetable\_cells, form\_groups) provides a clean reference for table relationships and foreign keys.

### Recommended First Schema to Scaffold

To follow the **AI Assembly-Line Strategy**, start with the **Pupil Core and Identity** table derived directly from the DfE CBDS:

* Primary identifier: Internal UUID
* Statutory identifiers: UPN (Unique Pupil Number - 13 chars with checksum algorithm), ULN (Unique Learner Number - 10 digits), NHS Number
* Demographics: Legal forename, legal surname, preferred name, DOB, sex, date of entry
* Characteristic flags: FSM (Free School Meals), Pupil Premium, SEN Provision (N, K, E), Looked After (LAC)

Would you like to generate the pure TypeScript Zod schema and Drizzle table definition for this core Pupil entity first?
