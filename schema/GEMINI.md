# SQLite Database Schemas (`schema`)

This directory contains the canonical SQL DDL definitions for the Optimis School MIS.

## Subdirectories

* [`schema/core`](file:///home/kieren/Documents/code/optimis/schema/core/GEMINI.md): Production 45-table `core.db` schema decomposed into 7 ordered domain SQL files (`00`–`06`) plus [`SCHEMA_MAP.md`](file:///home/kieren/Documents/code/optimis/schema/core/SCHEMA_MAP.md), executed automatically inside an atomic bootstrap transaction by [`dal.NewCoreDAL`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go).
