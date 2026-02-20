# Goyais MVP-1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deliver local-first AI coding assistant MVP with runtime SSE events, approval-based patching, SQLite persistence, and single-user sync backup.

**Architecture:** Tauri host communicates with Python runtime via localhost HTTP+SSE. Runtime is source-of-truth writer to SQLite. Sync server provides append-only push/pull for events.

**Tech Stack:** Tauri v2, React, FastAPI, LangGraph, Deep Agents, SQLite, Fastify.

---

This repository includes Step 1-7 baseline implementation and tests aligned with MVP scope.
