#!/usr/bin/env python3
"""Create Phasical tasks from docs/frontend-refactor-plan.yaml.

Phasical is Kaneo-based — API paths match the Kaneo reference:
https://kaneo.app/docs/api-reference/introduction

Base URL for Escalite Phasical: https://phasical.mdg-labs.dev/api
(this script uses host https://phasical.mdg-labs.dev with /api/* paths).

Conventions: epic parent + subtask leaves, blocks relations, Plan ID in descriptions.

Idempotent: skips tasks whose description contains a **Plan ID:** marker for that id.
Safe to re-run --apply to finish relations, labels, and status without duplicating tasks.
Always run --dry-run first.

Usage:
  export PHASICAL_API_KEY=...   # ephemeral key from operator
  export PHASICAL_BASE_URL=https://phasical.mdg-labs.dev  # optional
  python docs/import-frontend-refactor-plan.py --dry-run
  python docs/import-frontend-refactor-plan.py --apply
  python docs/import-frontend-refactor-plan.py --apply --epic fe-epic-shell
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml

ROOT = Path(__file__).resolve().parents[1]
PLAN_PATH = ROOT / "docs/frontend-refactor-plan.yaml"
DEFAULT_BASE_URL = "https://phasical.mdg-labs.dev"
PLAN_ID_RE = re.compile(r"\*\*Plan ID:?\*?\*?\s*`?([^*\n`]+)`?\*\*")
SIZE_TO_PRIORITY = {"S": "low", "M": "medium", "L": "high"}
DEFAULT_USER_ID = "K7YZpvbTOqMLBpqPGkwvccMlhGODm60D"


@dataclass
class PlannedTask:
    epic_id: str
    task_id: str
    title: str
    scope: str
    estimated_size: str
    priority: str
    labels: list[str]
    depends_on: list[str]
    acceptance_criteria: list[str]
    files: list[str]
    tests: list[str]
    implementation_notes: str = ""


@dataclass
class PlannedEpic:
    epic_id: str
    title: str
    description: str
    labels: list[str]
    suggested_order: int
    outline_refs: list[str] = field(default_factory=list)
    gap_section: str = ""
    tasks: list[PlannedTask] = field(default_factory=list)


class PhasicalClient:
    """Kaneo-compatible REST client (Phasical hosted instance)."""

    def __init__(self, base_url: str, api_key: str) -> None:
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key

    def _request(
        self,
        method: str,
        path: str,
        body: dict | None = None,
        *,
        ignore_http: set[int] | None = None,
    ) -> Any:
        url = f"{self.base_url}{path}"
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(
            url,
            data=data,
            method=method,
            headers={
                "Authorization": f"Bearer {self.api_key}",
                "Content-Type": "application/json",
                "Accept": "application/json",
            },
        )
        try:
            with urllib.request.urlopen(req, timeout=60) as resp:
                raw = resp.read().decode()
                return json.loads(raw) if raw else None
        except urllib.error.HTTPError as exc:
            if ignore_http and exc.code in ignore_http:
                return None
            detail = exc.read().decode()
            raise RuntimeError(f"{method} {path} failed ({exc.code}): {detail}") from exc

    def session(self) -> dict[str, Any]:
        return self._request("GET", "/api/auth/session")

    def list_all_tasks(self, project_id: str) -> list[dict[str, Any]]:
        """Fetch every task in the project (paginated at 100/page)."""
        by_id: dict[str, dict[str, Any]] = {}
        page = 1
        total_pages = 1
        while page <= total_pages:
            payload = self._request(
                "GET",
                f"/api/task/tasks/{project_id}?limit=100&page={page}",
            )
            data = payload.get("data", payload) if isinstance(payload, dict) else {}
            for column in data.get("columns", []):
                for task in column.get("tasks", []):
                    by_id[task["id"]] = task
            pagination = payload.get("pagination", {}) if isinstance(payload, dict) else {}
            total_pages = int(pagination.get("totalPages", page))
            page += 1
        return list(by_id.values())

    def get_task(self, task_id: str) -> dict[str, Any]:
        return self._request("GET", f"/api/task/{task_id}")

    def create_task(
        self,
        project_id: str,
        title: str,
        description: str,
        priority: str,
        status: str,
        user_id: str,
    ) -> dict[str, Any]:
        return self._request(
            "POST",
            f"/api/task/{project_id}",
            {
                "title": title,
                "description": description,
                "priority": priority,
                "status": status,
                "userId": user_id,
            },
        )

    def update_description(self, task_id: str, description: str) -> dict[str, Any]:
        return self._request(
            "PUT",
            f"/api/task/description/{task_id}",
            {"description": description},
        )

    def update_status(self, task_id: str, status: str) -> dict[str, Any]:
        return self._request(
            "PUT",
            f"/api/task/status/{task_id}",
            {"status": status},
        )

    def create_relation(self, source_task_id: str, target_task_id: str, relation_type: str) -> bool:
        """Return True if created, False if already existed (409)."""
        result = self._request(
            "POST",
            "/api/task-relation",
            {
                "sourceTaskId": source_task_id,
                "targetTaskId": target_task_id,
                "relationType": relation_type,
            },
            ignore_http={409},
        )
        return result is not None

    def create_task_label(
        self,
        workspace_id: str,
        name: str,
        task_id: str,
        color: str = "#64748b",
    ) -> dict[str, Any] | None:
        return self._request(
            "POST",
            "/api/label",
            {"name": name, "color": color, "workspaceId": workspace_id, "taskId": task_id},
            ignore_http={409},
        )


def load_env_local() -> None:
    env_path = ROOT / ".env.local"
    if not env_path.exists():
        return
    for line in env_path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, value = line.partition("=")
        os.environ.setdefault(key.strip(), value.strip().strip('"').strip("'"))


def plan_marker(plan_id: str) -> str:
    return f"**Plan ID: {plan_id}**"


def parse_plan_ids(tasks: list[dict[str, Any]]) -> dict[str, str]:
    mapping: dict[str, str] = {}
    for task in tasks:
        description = task.get("description") or ""
        for match in PLAN_ID_RE.finditer(description):
            plan_id = match.group(1).strip()
            if plan_id in mapping and mapping[plan_id] != task["id"]:
                raise ValueError(
                    f"Duplicate Plan ID {plan_id!r} on tasks "
                    f"{mapping[plan_id]} and {task['id']}"
                )
            mapping[plan_id] = task["id"]
    return mapping


def github_issue_number(task: dict[str, Any]) -> str | None:
    for link in task.get("externalLinks") or []:
        if link.get("resourceType") == "issue" and link.get("externalId"):
            return str(link["externalId"])
    return None


def build_github_map(tasks: list[dict[str, Any]]) -> dict[str, str]:
    github_by_task: dict[str, str] = {}
    for task in tasks:
        issue = github_issue_number(task)
        if issue:
            github_by_task[task["id"]] = issue
    return github_by_task


def task_label_names(task: dict[str, Any]) -> set[str]:
    return {label.get("name", "") for label in (task.get("labels") or []) if label.get("name")}


def ensure_task_label(
    client: PhasicalClient,
    task: dict[str, Any],
    workspace_id: str,
    label_name: str,
    *,
    color: str = "#64748b",
) -> bool:
    if label_name in task_label_names(task):
        return False
    client.create_task_label(workspace_id, label_name, task["id"], color)
    return True


def format_outline_section(outline_cfg: dict[str, Any], refs: list[str]) -> str:
    if not refs:
        return "_No Outline refs configured for this epic — see `docs/specs/README.md`._"
    docs = outline_cfg.get("docs", {})
    instruction = (outline_cfg.get("instruction") or "").strip()
    collection = outline_cfg.get("collection", "Escalite")
    lines = [
        f"**Collection:** {collection} (Outline MCP)",
        "",
        instruction,
        "",
        "**Documents to fetch before implementing:**",
    ]
    for ref in refs:
        doc = docs.get(ref, {})
        title = doc.get("title", ref)
        pointer = doc.get("pointer", f"docs/specs/{ref}.md")
        outline_id = doc.get("outline_id", "")
        id_line = f" — Outline doc id: `{outline_id}`" if outline_id else ""
        lines.append(f"- **{title}** (`{pointer}`){id_line}")
    return "\n".join(lines)


def format_gap_section(gap_section: str) -> str:
    if not gap_section:
        return "See [`docs/frontend-gap-analysis.md`](./frontend-gap-analysis.md) for audit context."
    return (
        f"Addresses gap analysis: **{gap_section}**. "
        "Full audit: [`docs/frontend-gap-analysis.md`](./frontend-gap-analysis.md)."
    )


def load_plan(path: Path, epic_filter: str | None) -> tuple[dict[str, Any], list[PlannedEpic]]:
    raw = yaml.safe_load(path.read_text(encoding="utf-8"))
    epic_outline_refs: dict[str, list[str]] = raw.get("epic_outline_refs", {})
    epic_gap_sections: dict[str, str] = raw.get("epic_gap_sections", {})
    epics: list[PlannedEpic] = []
    for epic_raw in raw.get("epics", []):
        epic_id = epic_raw["id"]
        if epic_filter and epic_id != epic_filter:
            continue
        tasks = []
        for t in epic_raw.get("tasks", []):
            size = t.get("estimated_size", "M")
            tasks.append(
                PlannedTask(
                    epic_id=epic_id,
                    task_id=t["id"],
                    title=t["title"],
                    scope=t.get("scope", "web"),
                    estimated_size=size,
                    priority=t.get("priority", SIZE_TO_PRIORITY.get(size, "medium")),
                    labels=t.get("labels", []),
                    depends_on=t.get("depends_on", []),
                    acceptance_criteria=t.get("acceptance_criteria", []),
                    files=t.get("files", []),
                    tests=t.get("tests", []),
                    implementation_notes=t.get("implementation_notes", ""),
                )
            )
        epics.append(
            PlannedEpic(
                epic_id=epic_id,
                title=epic_raw["title"],
                description=epic_raw.get("description", "").strip(),
                labels=epic_raw.get("labels", []),
                suggested_order=epic_raw.get("suggested_order", 99),
                outline_refs=epic_raw.get("outline_refs") or epic_outline_refs.get(epic_id, []),
                gap_section=epic_raw.get("gap_section") or epic_gap_sections.get(epic_id, ""),
                tasks=tasks,
            )
        )
    epics.sort(key=lambda e: e.suggested_order)
    return raw, epics


def build_leaf_description(
    task: PlannedTask,
    epic: PlannedEpic,
    outline_cfg: dict[str, Any],
) -> str:
    ac = "\n".join(f"- [ ] {item}" for item in task.acceptance_criteria)
    files = "\n".join(f"- `{f}`" for f in task.files) or "- _(see implementation notes)_"
    tests = "\n".join(f"- {t}" for t in task.tests) or "- Scoped CI per `.cursor/rules/06-local-ci-before-commit.mdc`"
    deps = ", ".join(f"`{d}`" for d in task.depends_on) if task.depends_on else "—"
    notes = task.implementation_notes.strip()
    notes_block = f"\n\n### Implementation notes\n\n{notes}" if notes else ""

    return f"""## {task.title}

{plan_marker(task.task_id)}

**Parent epic:** {epic.title} (`{epic.epic_id}`)

**Scope:** {task.scope} | **Size:** {task.estimated_size} | **Priority:** {task.priority}

**Depends on (plan IDs):** {deps}

---

### Gap analysis context

{format_gap_section(epic.gap_section)}

---

### Source of truth (Outline)

{format_outline_section(outline_cfg, epic.outline_refs)}

---

### Acceptance criteria

{ac}

---

### Files

{files}

---

### Tests

{tests}{notes_block}
"""


def build_epic_description_initial(epic: PlannedEpic, outline_cfg: dict[str, Any]) -> str:
    return f"""## Epic: {epic.title}

{plan_marker(f"epic:{epic.epic_id}")}

{epic.description}

---

### Gap analysis context

{format_gap_section(epic.gap_section)}

---

### Source of truth (Outline)

{format_outline_section(outline_cfg, epic.outline_refs)}

---

### Subtasks

_Subtask table updated after leaf tasks are created._

---

### Suggested implementation order

See `docs/frontend-refactor-plan.md` § Implementation order.
"""


def build_epic_description_final(
    epic: PlannedEpic,
    child_rows: list[tuple[PlannedTask, str | None]],
    outline_cfg: dict[str, Any],
) -> str:
    table_lines = []
    for task, issue_num in child_rows:
        gh = f" ([#{issue_num}](https://github.com/mdg-labs/escalite/issues/{issue_num}))" if issue_num else ""
        table_lines.append(f"| `{task.task_id}` | {task.title} | {task.scope} |{gh}")
    table = "\n".join(table_lines)
    return f"""## Epic: {epic.title}

{plan_marker(f"epic:{epic.epic_id}")}

{epic.description}

---

### Gap analysis context

{format_gap_section(epic.gap_section)}

---

### Source of truth (Outline)

{format_outline_section(outline_cfg, epic.outline_refs)}

---

### Subtasks

| Plan ID | Title | Scope |
| ------- | ----- | ----- |
{table}

---

### Suggested implementation order

See `docs/frontend-refactor-plan.md` § Implementation order.
"""


def patch_depends_section(
    description: str,
    depends_on: list[str],
    plan_to_id: dict[str, str],
    github_by_task: dict[str, str],
) -> str:
    if not depends_on:
        return description
    lines = ["**Depends on (GitHub):**"]
    for dep in depends_on:
        task_id = plan_to_id.get(dep)
        issue = github_by_task.get(task_id or "")
        if issue:
            lines.append(f"- #{issue} (`{dep}`)")
        else:
            lines.append(f"- `{dep}` (no GitHub issue yet)")
    block = "\n".join(lines)
    if "**Depends on (GitHub):**" in description:
        return re.sub(
            r"\*\*Depends on \(GitHub\):\*\*.*?(?=\n---|\n### |\Z)",
            block,
            description,
            flags=re.DOTALL,
        )
    return description.rstrip() + f"\n\n{block}\n"


def index_tasks_by_id(tasks: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    return {task["id"]: task for task in tasks}


def populate_id_maps(
    plan: list[PlannedEpic],
    issue_by_plan: dict[str, str],
) -> tuple[dict[str, str], dict[str, str]]:
    epic_ids: dict[str, str] = {}
    task_ids: dict[str, str] = {}
    for epic in plan:
        epic_key = f"epic:{epic.epic_id}"
        if epic_key in issue_by_plan:
            epic_ids[epic.epic_id] = issue_by_plan[epic_key]
        for task in epic.tasks:
            if task.task_id in issue_by_plan:
                task_ids[task.task_id] = issue_by_plan[task.task_id]
    return epic_ids, task_ids


def print_dry_run(
    plan: list[PlannedEpic],
    existing: dict[str, str],
    remote_count: int,
    project_id: str,
    ready_status: str,
    epic_label: str,
    plan_label: str,
) -> None:
    epic_count = len(plan)
    task_count = sum(len(e.tasks) for e in plan)
    skip_epics = sum(1 for e in plan if f"epic:{e.epic_id}" in existing)
    skip_tasks = sum(1 for e in plan for t in e.tasks if t.task_id in existing)

    print("=== DRY RUN — no Phasical API writes ===\n")
    print(f"Project: Escalite ({project_id})")
    print(f"Remote tasks (paginated): {remote_count}")
    print(f"Plan IDs matched: {len(existing)}")
    print(f"Would create: {epic_count - skip_epics} epics, {task_count - skip_tasks} leaf tasks")
    print(f"Would skip (existing Plan ID): {skip_epics} epics, {skip_tasks} leaf tasks")
    print(f"Would finalize: relations, labels (epic=`{epic_label}`, leaves=`{plan_label}`), status -> {ready_status}\n")

    for epic in plan:
        epic_key = f"epic:{epic.epic_id}"
        epic_status = "SKIP (exists)" if epic_key in existing else "CREATE"
        print(f"[{epic_status}] EPIC {epic_key}: {epic.title}")
        for task in epic.tasks:
            t_status = "SKIP (exists)" if task.task_id in existing else "CREATE"
            deps = ", ".join(task.depends_on) if task.depends_on else "none"
            print(f"  [{t_status}] {task.task_id}: {task.title}")
            print(f"           size={task.estimated_size} depends_on={deps}")
        print()


def apply_plan(
    client: PhasicalClient,
    plan: list[PlannedEpic],
    user_id: str,
    project_id: str,
    workspace_id: str,
    epic_label: str,
    plan_label: str,
    ready_status: str,
    outline_cfg: dict[str, Any],
) -> None:
    failures: list[str] = []
    created_epics = 0
    created_tasks = 0

    print("Loading existing tasks (paginated)...")
    remote_tasks = client.list_all_tasks(project_id)
    issue_by_plan = parse_plan_ids(remote_tasks)
    github_by_task = build_github_map(remote_tasks)
    tasks_by_id = index_tasks_by_id(remote_tasks)
    print(f"  found {len(remote_tasks)} tasks, {len(issue_by_plan)} plan IDs")

    epic_phasical_id, task_phasical_id = populate_id_maps(plan, issue_by_plan)
    skipped_epics = len(epic_phasical_id)
    skipped_tasks = len(task_phasical_id)

    print("Creating missing epics...")
    for epic in plan:
        key = f"epic:{epic.epic_id}"
        if key in issue_by_plan:
            print(f"  skip epic {key} (exists)")
            continue
        try:
            created = client.create_task(
                project_id,
                epic.title,
                build_epic_description_initial(epic, outline_cfg),
                "medium",
                "backlog",
                user_id,
            )
            issue_by_plan[key] = created["id"]
            epic_phasical_id[epic.epic_id] = created["id"]
            tasks_by_id[created["id"]] = created
            created_epics += 1
            print(f"  created epic {key} -> #{created.get('number', '?')}")
        except Exception as exc:
            failures.append(f"epic {key}: {exc}")

    print("Creating missing leaf tasks...")
    for epic in plan:
        for task in epic.tasks:
            if task.task_id in issue_by_plan:
                continue
            try:
                created = client.create_task(
                    project_id,
                    task.title,
                    build_leaf_description(task, epic, outline_cfg),
                    task.priority,
                    "backlog",
                    user_id,
                )
                issue_by_plan[task.task_id] = created["id"]
                task_phasical_id[task.task_id] = created["id"]
                tasks_by_id[created["id"]] = created
                created_tasks += 1
                if created_tasks % 20 == 0:
                    print(f"  ... {created_tasks} leaf tasks created so far")
            except Exception as exc:
                failures.append(f"task {task.task_id}: {exc}")

    if created_epics or created_tasks:
        print("Refreshing task list after creates...")
        remote_tasks = client.list_all_tasks(project_id)
        issue_by_plan = parse_plan_ids(remote_tasks)
        github_by_task = build_github_map(remote_tasks)
        tasks_by_id = index_tasks_by_id(remote_tasks)
        epic_phasical_id, task_phasical_id = populate_id_maps(plan, issue_by_plan)

    relations_created = 0
    relations_skipped = 0
    print("Wiring subtask relations (epic -> leaf)...")
    for epic in plan:
        parent_id = epic_phasical_id.get(epic.epic_id)
        if not parent_id:
            failures.append(f"subtask missing epic parent: {epic.epic_id}")
            continue
        for task in epic.tasks:
            child_id = task_phasical_id.get(task.task_id)
            if not child_id:
                failures.append(f"subtask missing child: {task.task_id}")
                continue
            try:
                if client.create_relation(parent_id, child_id, "subtask"):
                    relations_created += 1
                else:
                    relations_skipped += 1
            except Exception as exc:
                failures.append(f"subtask {epic.epic_id}->{task.task_id}: {exc}")

    print("Wiring blocks relations (depends_on)...")
    for epic in plan:
        for task in epic.tasks:
            child_id = task_phasical_id.get(task.task_id)
            if not child_id:
                continue
            for dep in task.depends_on:
                blocker_id = task_phasical_id.get(dep) or issue_by_plan.get(dep) or issue_by_plan.get(f"epic:{dep}")
                if not blocker_id:
                    failures.append(f"blocks {dep}->{task.task_id}: blocker not found")
                    continue
                try:
                    if client.create_relation(blocker_id, child_id, "blocks"):
                        relations_created += 1
                    else:
                        relations_skipped += 1
                except Exception as exc:
                    failures.append(f"blocks {dep}->{task.task_id}: {exc}")

    patched = 0
    print("Patching task descriptions with GitHub depends_on links...")
    for epic in plan:
        for task in epic.tasks:
            child_id = task_phasical_id.get(task.task_id)
            if not child_id or not task.depends_on:
                continue
            current = tasks_by_id.get(child_id) or client.get_task(child_id)
            description = current.get("description", "")
            updated = patch_depends_section(description, task.depends_on, issue_by_plan, github_by_task)
            if updated == description:
                continue
            try:
                client.update_description(child_id, updated)
                patched += 1
            except Exception as exc:
                failures.append(f"patch description {task.task_id}: {exc}")

    epics_patched = 0
    print("Updating epic descriptions with subtask tables...")
    for epic in plan:
        parent_id = epic_phasical_id.get(epic.epic_id)
        if not parent_id:
            continue
        child_rows: list[tuple[PlannedTask, str | None]] = []
        for task in epic.tasks:
            child_id = task_phasical_id.get(task.task_id)
            issue_num = github_by_task.get(child_id or "") if child_id else None
            child_rows.append((task, issue_num))
        final_desc = build_epic_description_final(epic, child_rows, outline_cfg)
        current = tasks_by_id.get(parent_id) or client.get_task(parent_id)
        if current.get("description") == final_desc:
            continue
        try:
            client.update_description(parent_id, final_desc)
            epics_patched += 1
        except Exception as exc:
            failures.append(f"patch epic {epic.epic_id}: {exc}")

    labels_attached = 0
    print("Attaching per-task labels...")
    for epic in plan:
        parent_id = epic_phasical_id.get(epic.epic_id)
        if parent_id:
            current = tasks_by_id.get(parent_id) or client.get_task(parent_id)
            for label_name in {epic_label, plan_label, *epic.labels}:
                try:
                    if ensure_task_label(client, current, workspace_id, label_name):
                        labels_attached += 1
                        current.setdefault("labels", []).append({"name": label_name})
                except Exception as exc:
                    failures.append(f"label {label_name} on epic {epic.epic_id}: {exc}")

        for task in epic.tasks:
            child_id = task_phasical_id.get(task.task_id)
            if not child_id:
                continue
            current = tasks_by_id.get(child_id) or client.get_task(child_id)
            attach_names = set(task.labels) | {plan_label}
            for label_name in attach_names:
                try:
                    if ensure_task_label(client, current, workspace_id, label_name):
                        labels_attached += 1
                        current.setdefault("labels", []).append({"name": label_name})
                except Exception as exc:
                    failures.append(f"label {label_name} on {task.task_id}: {exc}")

    status_updated = 0
    print(f"Setting status -> {ready_status} where not already...")
    all_ids = list({*task_phasical_id.values(), *epic_phasical_id.values()})
    for phasical_id in all_ids:
        current = tasks_by_id.get(phasical_id) or client.get_task(phasical_id)
        if current.get("status") == ready_status:
            continue
        try:
            client.update_status(phasical_id, ready_status)
            status_updated += 1
        except Exception as exc:
            failures.append(f"status {phasical_id}: {exc}")

    print("\n=== APPLY SUMMARY ===")
    print(f"Remote tasks: {len(remote_tasks)}")
    print(f"Created epics: {created_epics} (pre-existing epics: {skipped_epics})")
    print(f"Created leaf tasks: {created_tasks} (pre-existing leaves: {skipped_tasks})")
    print(f"Relations: {relations_created} created, {relations_skipped} already existed")
    print(f"Descriptions patched: {patched} leaves, {epics_patched} epics")
    print(f"Labels attached: {labels_attached}")
    print(f"Status moved to {ready_status}: {status_updated}")
    print(f"Failures: {len(failures)}")
    for failure in failures[:20]:
        print(f"  - {failure}")
    if len(failures) > 20:
        print(f"  ... and {len(failures) - 20} more")


def resolve_user_id(client: PhasicalClient) -> str:
    env_user = os.environ.get("PHASICAL_USER_ID")
    if env_user:
        return env_user
    try:
        session = client.session()
        user = session.get("user") or session
        user_id = user.get("id")
        if user_id:
            return str(user_id)
    except Exception:
        pass
    return DEFAULT_USER_ID


def main() -> int:
    load_env_local()
    parser = argparse.ArgumentParser(description="Import frontend-refactor plan into Phasical")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--dry-run", action="store_true", help="Print planned creates without API writes")
    group.add_argument("--apply", action="store_true", help="Create/finalize tasks in Phasical")
    parser.add_argument("--epic", help="Import only one epic id (e.g. fe-epic-shell)")
    parser.add_argument(
        "--base-url",
        default=os.environ.get("PHASICAL_BASE_URL", DEFAULT_BASE_URL),
        help="Phasical host (default: https://phasical.mdg-labs.dev); paths use /api/*",
    )
    args = parser.parse_args()

    api_key = os.environ.get("PHASICAL_API_KEY")
    if not api_key:
        print("ERROR: PHASICAL_API_KEY is required (env or .env.local)", file=sys.stderr)
        return 1

    config, plan = load_plan(PLAN_PATH, args.epic)
    if args.epic and not plan:
        print(f"ERROR: Epic not found: {args.epic}", file=sys.stderr)
        return 1

    phasical_cfg = config.get("phasical", {})
    project_id = phasical_cfg["project_id"]
    workspace_id = phasical_cfg["workspace_id"]
    epic_label = phasical_cfg.get("epic_label", "epic")
    plan_label = phasical_cfg.get("plan_label", "frontend-refactor")
    ready_status = phasical_cfg.get("ready_status", "ready")
    outline_cfg = config.get("outline", {})

    client = PhasicalClient(args.base_url, api_key)
    remote_tasks = client.list_all_tasks(project_id)
    existing = parse_plan_ids(remote_tasks)

    if args.dry_run:
        print_dry_run(
            plan, existing, len(remote_tasks), project_id, ready_status, epic_label, plan_label
        )
        return 0

    user_id = resolve_user_id(client)
    apply_plan(
        client,
        plan,
        user_id,
        project_id,
        workspace_id,
        epic_label,
        plan_label,
        ready_status,
        outline_cfg,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
