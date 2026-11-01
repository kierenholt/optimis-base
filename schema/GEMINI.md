# SQLite Database Schemas (`schema`)

This directory contains the canonical SQL DDL definitions for the Optimis School MIS.

## Subdirectories

* [`schema/core`](file:///home/kieren/Documents/code/optimis/schema/core/GEMINI.md): Production 46-table `core.db` schema decomposed into 7 ordered domain SQL files (`00`–`06`), executed automatically inside an atomic bootstrap transaction by [`dal.NewCoreDAL`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go).
