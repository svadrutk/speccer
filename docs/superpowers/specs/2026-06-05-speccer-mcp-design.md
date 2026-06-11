# Speccer MCP: Concurrent Architecture & Tech Spec Validator

**A Model Context Protocol (MCP) Server for Auditing, Verifying, and Scaffolding AI-Generated Specs**

---

## 1. Product Overview & Core Philosophy

**Speccer MCP** is a lightweight, high-performance Go-based Model Context Protocol server. It is built to bridge the gap between AI-generated engineering specs and actual workspace codebases. 

In modern development workflows (e.g., Cursor, Claude Code, OpenCode), feature specifications are increasingly drafted by LLMs. While structurally complete, these specifications often suffer from:
1. **Architectural Hand-waving:** Statements like *"the system will handle backpressure automatically"* without specifying queue sizes, dead-letter queues, or retry behaviors.
2. **Data-Model & Schema Inconsistencies:** Out-of-sync UUID/Integer IDs, missing foreign-key indexes, or mismatched table keys between different sections of the spec.
3. **Spec-to-Code Drift:** Over time, the actual workspace database migrations, APIs, and application models diverge from the written specification, turning it into stale "shelfware".

**Speccer MCP** acts as an active compiler and verifier. It parses arbitrary specs (prose or structured) and workspace files into ASTs, cross-references them, detects structural drift, and scaffolds boilerplate implementation directly. 

**MVP Focus:** To guarantee extreme robustness and reliability, the Speccer MVP focuses specifically on **Python** workspaces (analyzing SQLAlchemy/SQLModel models, Django models, and FastAPI/Flask routes) using a configuration-driven, hybrid parsing strategy.

---

## 2. System Architecture

By conforming purely to the **Model Context Protocol (MCP)** using the **Stdio Transport**, Speccer is universally compatible with **Cursor, Claude Code, OpenCode, and Claude Desktop**.

```
┌────────────────────────────────────────────────────────┐
│                        CLIENT                          │
│    (Cursor / Claude Code / OpenCode / Claude Desktop)  │
└───────────────────────────┬────────────────────────────┘
                            │
                            │ stdin / stdout (JSON-RPC 2.0 over Stdio)
                            ▼
┌────────────────────────────────────────────────────────┐
│               GO MCP SERVER (SPECCER)                  │
│                                                        │
│   - yuin/goldmark (Markdown Spec AST Parsing)          │
│   - Semantic Router & Classifier (Prose Tagging)       │
│   - File Drift Engine (Go-based Structural Diffing)    │
│   - Scaffolding Compiler (SQL, Python, Code Generator) │
└───────────────────────────┬────────────────────────────┘
                            │ Calls local python helper
                            ▼
┌────────────────────────────────────────────────────────┐
│             PYTHON PARSER HELPER (SPECCER)             │
│                                                        │
│   - Native ast module (SQLAlchemy, FastAPI Parsing)    │
│   - Generates standardized JSON IR                     │
└───────────────────────────┬────────────────────────────┘
                            │ Reads & Diffs
                            ▼
┌────────────────────────────────────────────────────────┐
│                  DEVELOPER WORKSPACE                   │
│                                                        │
│   Reads:  feature-spec.md, speccer.json                │
│   Diffs:  /models.py, /routes.py, /db/migrations       │
│   Writes: /models.py, new_migration.sql (scaffold)     │
└────────────────────────────────────────────────────────┘
```

---

## 3. The Semantic AST Compiler & Routing Pipeline

Because AI-generated specs have highly variable, non-standard layouts, Speccer does not rely on hardcoded section headers. Instead, it utilizes a two-tier **Semantic AST Compiler**:

1. **AST Chunking:** Speccer uses `yuin/goldmark` to compile the raw Markdown into block-level elements (headings, list items, prose paragraphs, and code fences).
2. **Local & Cached Semantic Classifier:** For each heading and its accompanying prose, Speccer extracts the heading label and the first 2-3 sentences. To assign **Intent Domains**, it employs:
   * **Primary Local Heuristic:** A fast local regex classifier mapping keywords directly to intent domains.
   * **Optional LLM Classification Fallback:** If high-precision classification is needed for ambiguous headings, Speccer can trigger a cached LLM call. This pass is strictly **opt-in** (requiring a developer-provided API key like `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` in the environment) and results are highly cached by block SHA-256. If no keys are present, Speccer defaults to local heuristics, maintaining absolute sandboxing and zero network egress by default.
   Intent domains map as follows:
   * `# Database`, `# Storage`, `# Entities`, or prose about tables → `data_schema`
   * `# Infrastructure`, `# Deployment`, `# Docker`, or prose about servers → `infra_config`
   * `# Overview`, `# Core Flow`, or prose about business logic → `system_prose`
3. **Targeted Routing:** AST chunks are routed to domain-specific validators based on these tags. This keeps validations extremely fast and cost-effective.

---

## 4. Exposed MCP Protocol Specification

The server exposes the following JSON-RPC 2.0 tools to the host:

### 1. `audit_specification`
Audits prose or structured specs for architectural omissions, logical flaws, and database schema issues.

* **Arguments:**
  * `spec_path` (string, required): Absolute path to the Markdown specification file.
* **Analysis Routine:**
  * **Prose Omission Checker:** Searches for "magical scaling" or "hand-waving" phrases and tags missing requirements (e.g., rate limits, retry limits, circuit breakers, dead-letter queues).
  * **Schema Inconsistency Checker:** Cross-references table definitions across different sections of the spec. Warns on mismatched foreign key types (e.g., `UUID` vs `INT`), missing unique indexes on query fields, or case drift (`userID` vs `user_id`).

### 2. `detect_drift`
Verifies whether the actual workspace code/schema matches the written specification.

* **Arguments:**
  * `spec_path` (string, required): Absolute path to the specification file.
  * `workspace_path` (string, required): Absolute path to the project root directory.
* **Analysis Routine:**
  * **Database Drift:** Parses SQL, SQLAlchemy, or declarative schema code fences in the spec and diffs them against defined Python database models (e.g., SQLAlchemy `Base` classes, SQLModel classes, or Django models) located in the paths specified by `speccer.json`.
  * **API Drift:** Parses REST endpoint contracts (e.g. paths, query/body parameters, return codes) in the spec and matches them against routes defined in pythonic router files (e.g., FastAPI endpoints, Flask route decorators, Django views).

### 3. `scaffold_code`
Translates spec definitions (SQL tables, API contracts, infrastructure configurations) into ready-to-use workspace boilerplate.

* **Arguments:**
  * `spec_path` (string, required): Absolute path to the specification file.
  * `output_dir` (string, required): Absolute path to write the generated files.
* **Scaffolding Routine:**
  * **Config Resolution:** Reads the project's root `speccer.json` configuration to automatically resolve the target database engine (e.g., SQLAlchemy, SQLModel, or Django) and API framework (e.g., FastAPI, Flask, or Django).
  * **Generators:**
    * Generates Python database model declarations (SQLAlchemy or SQLModel) matching the configured `database.engine` directly from spec schemas.
    * Creates boilerplate controller files and endpoint stub handlers matching the configured `api.framework`.
    * Generates standard Dockerfiles and Compose configs from parsed infra requirements.

### 4. `initialize_workspace`
Scans the local repository and dynamically generates a draft `speccer.json` configuration using fast heuristic candidate discovery.

* **Arguments:**
  * `workspace_path` (string, required): Absolute path to the project root directory.
* **Initialization Routine:**
  * **File Scan & Heuristics:** Searches the workspace for Python files containing database-related classes or imports (e.g., SQLAlchemy, SQLModel, Pydantic, Django Models) and API-related frameworks or decorators (e.g., FastAPI, Flask, Django Views).
  * **Draft Assembly:** Groups candidate files into recommended glob patterns or paths.
  * **Return Content:** Returns a JSON object with the recommended `speccer.json` structure, plus lists of discovered candidate files for LLM/user refinement.

### 5. `write_workspace_config`
Writes the finalized or LLM-refined `speccer.json` configuration to the root of the workspace.

* **Arguments:**
  * `workspace_path` (string, required): Absolute path to the project root directory.
  * `config` (object, required): The validated `speccer.json` payload.
* **Writing Routine:**
  * Performs JSON schema validation on the incoming config payload.
  * Safely writes the `speccer.json` file to the root of the workspace directory.

---

## 5. Bounded Drift Detection & Diff Engine Algorithm

To perform extremely robust drift analysis without language-parsing complexity in compiled Go, Speccer employs a **Configuration-Driven Hybrid Architecture** focused on Python targets for its MVP.

### 5.1 MVP Scope, Heuristic Boundaries & Graceful Failures
To guarantee extreme robustness and prevent fragile "magic" detection behaviors, Speccer enforces strict, explicit boundaries on what it supports:
1. **Supported Stack (The Sweet Spot):**
   * **Database Engines:** standard SQLAlchemy (Declarative Base), SQLModel, and standard Django models.
   * **API Frameworks:** FastAPI (using Pydantic models), standard Flask (route decorators), and standard Django views.
2. **Detection & Initialization Failure Bounds:**
   * If the heuristic scanner in `initialize_workspace` finds zero standard markers (e.g., no imports of `FastAPI`, `APIRouter`, `SQLAlchemy`, `SQLModel`, `BaseModel`, or `django.db`), it will not generate a guessed draft configuration.
   * Instead, it gracefully aborts with a structured return message: `"No standard FastAPI/SQLAlchemy/Django patterns detected. Speccer auto-detection supports standard structures. You can still manually configure speccer.json to map your custom files."`
3. **Unresolved Types:**
   * Custom frameworks, highly dynamic code, or dynamic meta-programming elements are treated as "unresolved external models/types" rather than causing parser crashes.

### 5.2 Configuration Boundaries (`speccer.json`)
Workspace drift checks must be configured via a root-level `speccer.json` file. This prevents fragile full-directory scanning and establishes exact mapping scopes. Upon server startup or tool invocation, Speccer performs strict JSON schema validation on `speccer.json` to ensure required fields and correct frameworks are specified, failing early with clean error messages if misconfigured. To minimize manual updates, paths in `"source_files"` and `"pydantic_files"` support standard glob patterns (e.g., `**/*.py`):

```json
{
  "spec_path": "docs/specs/user-management.md",
  "database": {
    "engine": "sqlalchemy",
    "source_files": ["app/models/**/*.py"]
  },
  "api": {
    "framework": "fastapi",
    "source_files": ["app/routers/**/*.py"],
    "pydantic_files": ["app/schemas/**/*.py"]
  }
}
```

### 5.3 Go & Python Hybrid Parse Pipeline
Instead of compiling tree-sitter or complex lexers inside Go to read arbitrary Python code:
1. **Spec Parser:** The Go server parses Markdown tables, JSON payloads, or declarative schema code blocks in the spec to construct the expected structural maps.
2. **Interpreter & Virtualenv Auto-Detection:** To ensure compatibility with custom workspace dependencies, the Go server automatically scans the workspace root for active Python virtual environments (e.g., `.venv`, `venv`, `env`, or queries poetry/pipenv/hatch if active) and executes the Python parser using the discovered local interpreter context, falling back to system-level `python3` only if none are found.
3. **Workspace Python Parser Invocation:** The Go server resolves and deduplicates all glob patterns to prevent redundant AST parsing, and invokes the bundled Python AST helper (e.g., `python3 -m speccer_parser --files <paths> --framework <engine> --output-file <temp_path>`). To completely avoid issues with standard output pollution from deprecation warnings, third-party library imports, or user print statements, the Go server commands the helper to output logs/warnings to standard error (`sys.stderr`) and write the clean JSON IR exclusively to a temporary file path or wrapped in distinct stream markers (`===SPECCER-JSON-START===`).
4. **AST Extraction & Cross-File Symbol Table (Database & Schemas):**
   * **Parsing Engine:** The helper script parses the targeted Python files using Python's native `ast` library.
   * **Cross-File Import Resolution:** To resolve schemas imported from other modules (e.g., `from .schemas import UserBase`) without executing unsafe workspace code, the helper first constructs a workspace-wide **Static Symbol Table** by scanning all files matched by the glob patterns in `speccer.json`. It maps imported names to their source AST definitions, allowing it to resolve nested Pydantic models and SQLAlchemy mixins across multiple files purely statically.
   * **Property/Constraint Extraction:** For Pydantic models (classes subclassing `BaseModel`), it resolves python type hints (e.g., `str`, `Optional[int]`, list annotations) and validations (e.g., `gt=0`, `max_length=20`) from the AST nodes.
5. **Intermediate Representation (IR) Exchange:** The Go server reads the clean JSON schema map (IR) from the specified temporary file or stdout stream marker.

### 5.4 Comparative Diff Structure

```go
type Field struct {
    Name     string `json:"name"`
    Type     string `json:"type"`     // Unified normalized type (e.g., "string", "int", "uuid")
    Nullable bool   `json:"nullable"`
}

type ModelSchema struct {
    Name   string           `json:"name"`
    Fields map[string]Field `json:"fields"`
}

type APIEndpoint struct {
    Path         string           `json:"path"`
    Method       string           `json:"method"` // GET, POST, etc.
    Params       []Field          `json:"params"`
    RequestBody  map[string]Field `json:"request_body,omitempty"`  // Extracted Pydantic request shape
    ResponseBody map[string]Field `json:"response_body,omitempty"` // Extracted Pydantic response shape
}
```

**Comparison Rules:**
* **Missing Elements:** Detects models, payload fields, or endpoint paths defined in the spec but missing in the Python codebase.
* **Property Mismatch:** Detects field types mismatch (e.g., column is designated `UUID` in spec, but is a `String` or `Integer` in SQLAlchemy).
* **Nullability Check:** Flags database constraints mismatch (e.g. `nullable=False` in code but optional in spec).
* **Pydantic Schema Payload Audit:** Cross-references JSON request/response structures from the Markdown spec code blocks directly against the fields and properties defined on the FastAPI endpoint's associated Pydantic models.
* **API Signature Audit:** Warns when query/body parameters in the specification do not match the parsed path parameters or input variables in the python routes.

---

## 6. Security & Sandboxing Boundaries

* **Read Limits:** Speccer only reads files within the provided workspace root. It rejects path-traversal inputs containing `..`.
* **Write Confirmation:** The `scaffold_code` tool never overwrites active files unless explicitly authorized or if the host editor displays a confirmation dialogue (e.g., Cursor's file-write preview).
* **Local First:** All AST parsing, schema extraction, and drift diffing run 100% locally in compiled Go, ensuring zero network egress for workspace code.
