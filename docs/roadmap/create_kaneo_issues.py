#!/usr/bin/env python3
"""Create Kaneo issues from docs/roadmap/roadmap.yaml.

Uses the Kaneo REST API (same paths as the user-kaneo MCP server). Follows
conventions from .agents/skills/kaneo-intake/SKILL.md (epic parent + subtask
leaves, blocks relations, Roadmap ID in descriptions).

Idempotent: skips issues whose description contains ``**Roadmap ID: <id>**``.
Safe to re-run --apply to finish relations, labels, and status without
duplicating issues. Always run --dry-run first.

Usage:
  export KANEO_API_KEY=...   # or place in untracked .env.local (not committed)
  python docs/roadmap/create_kaneo_issues.py --dry-run
  python docs/roadmap/create_kaneo_issues.py --apply
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

ROOT = Path(__file__).resolve().parents[2]
ROADMAP_PATH = ROOT / "docs/roadmap/roadmap.yaml"
DEFAULT_BASE_URL = "https://kaneo.mdg-labs.dev"
PROJECT_ID = "bkbnmftqdr54r9gcgrgtndbw"
WORKSPACE_ID = "X3VbytvC7pKgazK2dAsOQIFtdGYRzdGH"
GITHUB_REPO = "mdg-labs/escalite"
READY_STATUS = "ready"
DEFAULT_USER_ID = "K7YZpvbTOqMLBpqPGkwvccMlhGODm60D"
EPIC_LABEL = "epic"

SIZE_TO_PRIORITY = {"S": "low", "M": "medium", "L": "high"}
ROADMAP_ID_RE = re.compile(r"\*\*Roadmap ID:\s*([^*\n]+)\*\*")
GITHUB_DEPENDS_RE = re.compile(r"\*\*Depends on:\*\*", re.MULTILINE)


@dataclass
class PlannedEpic:
    phase_id: str
    phase_name: str
    epic_id: str
    title: str
    description: str
    spec_refs: list[str]
    tasks: list[PlannedTask] = field(default_factory=list)


@dataclass
class PlannedTask:
    phase_id: str
    epic_id: str
    task_id: str
    title: str
    description: str
    depends_on: list[str]
    labels: list[str]
    estimated_size: str
    external_dependency: dict[str, Any] | None


class KaneoClient:
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

    def list_all_tasks(self, project_id: str) -> list[dict[str, Any]]:
        """Fetch every task in the project (Kaneo paginates at 100/page)."""
        by_id: dict[str, dict[str, Any]] = {}
        page = 1
        total_pages = 1
        while page <= total_pages:
            payload = self._request(
                "GET",
                f"/api/task/tasks/{project_id}?limit=100&page={page}",
            )
            for column in payload.get("data", {}).get("columns", []):
                for task in column.get("tasks", []):
                    by_id[task["id"]] = task
            pagination = payload.get("pagination", {})
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
    ) -> dict[str, Any]:
        return self._request(
            "POST",
            "/api/label",
            {"name": name, "color": color, "workspaceId": workspace_id, "taskId": task_id},
            ignore_http={409},
        )


def ensure_task_label(
    client: KaneoClient,
    task: dict[str, Any],
    label_name: str,
    *,
    color: str = "#64748b",
) -> bool:
    """Create a per-task label if missing. Returns True when created."""
    if label_name in task_label_names(task):
        return False
    client.create_task_label(WORKSPACE_ID, label_name, task["id"], color)
    return True


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


def roadmap_marker(roadmap_id: str) -> str:
    return f"**Roadmap ID: {roadmap_id}**"


def parse_roadmap_ids(tasks: list[dict[str, Any]]) -> dict[str, str]:
    mapping: dict[str, str] = {}
    for task in tasks:
        description = task.get("description") or ""
        for match in ROADMAP_ID_RE.finditer(description):
            roadmap_id = match.group(1).strip()
            if roadmap_id in mapping and mapping[roadmap_id] != task["id"]:
                raise ValueError(
                    f"Duplicate Roadmap ID {roadmap_id!r} on tasks "
                    f"{mapping[roadmap_id]} and {task['id']}"
                )
            mapping[roadmap_id] = task["id"]
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


def format_spec_refs(spec_refs: list[str]) -> str:
    if not spec_refs:
        return "_none_"
    return "\n".join(
        f"- `docs/specs/{ref.split('#')[0]}.md`" + (f" ({ref})" if "#" in ref else "")
        for ref in spec_refs
    )


def build_task_description(task: dict[str, Any], phase: dict[str, Any], epic: dict[str, Any], parent_marker: str) -> str:
    ext = task.get("external_dependency")
    ext_section = ""
    if ext:
        blocked = f"\n**Blocked by task:** `{ext['blocked_by']}`" if ext.get("blocked_by") else ""
        ext_section = (
            "\n\n### External dependency\n\n"
            f"⚠️ **Requires human action:** {ext['reason']}{blocked}\n"
        )

    depends = task.get("depends_on") or []
    depends_section = "\n".join(f"- `{dep}`" for dep in depends) if depends else "_none_"

    ac = "\n".join(f"- [ ] {item}" for item in task.get("acceptance_criteria", []))
    dod = "\n".join(f"- [ ] {item}" for item in task.get("definition_of_done", []))
    labels = ", ".join(f"`{label}`" for label in task.get("labels", []))

    return f"""## {task['title']}

**Parent epic:** {parent_marker}
**Phase:** {phase['id']} — {phase['name']}
{roadmap_marker(task['id'])}

**Depends on (roadmap):**
{depends_section}

**Estimated size:** {task.get('estimated_size', 'M')}
**Labels:** {labels}
{ext_section}
---

### Description

{task['description']}

---

### Acceptance criteria

{ac}

---

### Definition of done

{dod}

---

### Spec references

{format_spec_refs(task.get('spec_refs', []))}
"""


def build_epic_description(epic: dict[str, Any], phase: dict[str, Any]) -> str:
    return f"""## Epic: {epic['title']}

**Phase:** {phase['id']} — {phase['name']}
{roadmap_marker(f"epic:{epic['id']}")}

{epic['description']}

---

### Spec references

{format_spec_refs(epic.get('spec_refs', []))}

---

### Subtasks

_(Populated after child issues are created.)_
"""


def build_epic_description_final(
    epic: PlannedEpic,
    child_issues: list[tuple[PlannedTask, str | None]],
) -> str:
    rows = []
    for child, issue_num in child_issues:
        if issue_num:
            link = f"[#{issue_num}](https://github.com/{GITHUB_REPO}/issues/{issue_num})"
        else:
            link = "_(pending GitHub sync)_"
        rows.append(f"| {link} | `{child.task_id}` | {child.title} |")
    table = "\n".join(rows) if rows else "_none_"
    return f"""## Epic: {epic.title}

**Phase:** {epic.phase_id} — {epic.phase_name}
{roadmap_marker(f"epic:{epic.epic_id}")}

{epic.description}

---

### Spec references

{format_spec_refs(epic.spec_refs)}

---

### Subtasks

| Issue | Roadmap ID | Title |
| ----- | ---------- | ----- |
{table}
"""


def patch_depends_section(
    description: str,
    depends_on: list[str],
    issue_by_roadmap: dict[str, str],
    github_by_task: dict[str, str],
) -> str:
    if not depends_on:
        return description
    if GITHUB_DEPENDS_RE.search(description):
        return description
    lines = ["**Depends on:**"]
    for dep in depends_on:
        kaneo_id = issue_by_roadmap.get(dep)
        issue = github_by_task.get(kaneo_id or "")
        if issue:
            lines.append(
                f"- [#{issue}](https://github.com/{GITHUB_REPO}/issues/{issue}) (`{dep}`)"
            )
        else:
            lines.append(f"- `{dep}` _(GitHub link pending)_")
    block = "\n".join(lines)
    return re.sub(
        r"\*\*Depends on \(roadmap\):\*\*\n(?:- .+\n?|_none_\n?)+",
        block + "\n\n**Depends on (roadmap):**\n" + "\n".join(f"- `{d}`" for d in depends_on) + "\n",
        description,
        count=1,
    )


def plan_from_roadmap(roadmap: dict[str, Any]) -> list[PlannedEpic]:
    planned: list[PlannedEpic] = []
    for phase in roadmap["phases"]:
        for epic in phase["epics"]:
            pe = PlannedEpic(
                phase_id=phase["id"],
                phase_name=phase["name"],
                epic_id=epic["id"],
                title=epic["title"],
                description=epic["description"],
                spec_refs=epic.get("spec_refs", []),
            )
            for task in epic["tasks"]:
                pe.tasks.append(
                    PlannedTask(
                        phase_id=phase["id"],
                        epic_id=epic["id"],
                        task_id=task["id"],
                        title=task["title"],
                        description=build_task_description(task, phase, epic, f"`epic:{epic['id']}`"),
                        depends_on=task.get("depends_on", []),
                        labels=list(task.get("labels", [])),
                        estimated_size=task.get("estimated_size", "M"),
                        external_dependency=task.get("external_dependency"),
                    )
                )
            planned.append(pe)
    return planned


def index_tasks_by_kaneo_id(tasks: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    return {task["id"]: task for task in tasks}


def populate_id_maps(
    plan: list[PlannedEpic],
    issue_by_roadmap: dict[str, str],
) -> tuple[dict[str, str], dict[str, str]]:
    epic_kaneo_id: dict[str, str] = {}
    task_kaneo_id: dict[str, str] = {}
    for epic in plan:
        epic_key = f"epic:{epic.epic_id}"
        if epic_key in issue_by_roadmap:
            epic_kaneo_id[epic.epic_id] = issue_by_roadmap[epic_key]
        for task in epic.tasks:
            if task.task_id in issue_by_roadmap:
                task_kaneo_id[task.task_id] = issue_by_roadmap[task.task_id]
    return epic_kaneo_id, task_kaneo_id


def print_dry_run(plan: list[PlannedEpic], existing: dict[str, str], remote_count: int) -> None:
    epic_count = len(plan)
    task_count = sum(len(e.tasks) for e in plan)
    skip_epics = sum(1 for e in plan if f"epic:{e.epic_id}" in existing)
    skip_tasks = sum(1 for e in plan for t in e.tasks if t.task_id in existing)

    print("=== DRY RUN — no Kaneo API writes ===\n")
    print(f"Project: Escalite ({PROJECT_ID})")
    print(f"Remote tasks (paginated): {remote_count}")
    print(f"Roadmap IDs matched: {len(existing)}")
    print(f"Would create: {epic_count - skip_epics} epics, {task_count - skip_tasks} leaf tasks")
    print(f"Would skip (existing Roadmap ID): {skip_epics} epics, {skip_tasks} leaf tasks")
    print(f"Would finalize: relations, labels (incl. {EPIC_LABEL} on epics), status -> {READY_STATUS}\n")

    for epic in plan:
        epic_key = f"epic:{epic.epic_id}"
        epic_status = "SKIP (exists)" if epic_key in existing else "CREATE"
        print(f"[{epic_status}] EPIC {epic_key}: {epic.title}")
        print(f"  phase: {epic.phase_id} — {epic.phase_name}")
        for task in epic.tasks:
            t_status = "SKIP (exists)" if task.task_id in existing else "CREATE"
            ext = " ⚠️ EXTERNAL" if task.external_dependency else ""
            deps = ", ".join(task.depends_on) if task.depends_on else "none"
            print(f"  [{t_status}] {task.task_id}: {task.title}{ext}")
            print(f"           size={task.estimated_size} depends_on={deps} labels={','.join(task.labels)}")
        print()


def apply_plan(client: KaneoClient, plan: list[PlannedEpic], user_id: str) -> None:
    failures: list[str] = []
    created_epics = 0
    created_tasks = 0

    print("Loading existing tasks (paginated)...")
    remote_tasks = client.list_all_tasks(PROJECT_ID)
    issue_by_roadmap = parse_roadmap_ids(remote_tasks)
    github_by_task = build_github_map(remote_tasks)
    tasks_by_id = index_tasks_by_kaneo_id(remote_tasks)
    print(f"  found {len(remote_tasks)} tasks, {len(issue_by_roadmap)} roadmap IDs")

    epic_kaneo_id, task_kaneo_id = populate_id_maps(plan, issue_by_roadmap)
    skipped_epics = len(epic_kaneo_id)
    skipped_tasks = len(task_kaneo_id)

    print("Creating missing epics...")
    for epic in plan:
        key = f"epic:{epic.epic_id}"
        if key in issue_by_roadmap:
            print(f"  skip epic {key} (exists)")
            continue
        try:
            created = client.create_task(
                PROJECT_ID,
                epic.title,
                build_epic_description(
                    {
                        "title": epic.title,
                        "description": epic.description,
                        "id": epic.epic_id,
                        "spec_refs": epic.spec_refs,
                    },
                    {"id": epic.phase_id, "name": epic.phase_name},
                ),
                "medium",
                "backlog",
                user_id,
            )
            issue_by_roadmap[key] = created["id"]
            epic_kaneo_id[epic.epic_id] = created["id"]
            tasks_by_id[created["id"]] = created
            created_epics += 1
            print(f"  created epic {key} -> #{created.get('number', '?')}")
        except Exception as exc:
            failures.append(f"epic {key}: {exc}")

    print("Creating missing leaf tasks...")
    for epic in plan:
        for task in epic.tasks:
            if task.task_id in issue_by_roadmap:
                continue
            try:
                priority = SIZE_TO_PRIORITY.get(task.estimated_size, "medium")
                created = client.create_task(
                    PROJECT_ID,
                    task.title,
                    task.description,
                    priority,
                    "backlog",
                    user_id,
                )
                issue_by_roadmap[task.task_id] = created["id"]
                task_kaneo_id[task.task_id] = created["id"]
                tasks_by_id[created["id"]] = created
                created_tasks += 1
                if created_tasks % 20 == 0:
                    print(f"  ... {created_tasks} leaf tasks created so far")
            except Exception as exc:
                failures.append(f"task {task.task_id}: {exc}")

    if created_epics or created_tasks:
        print("Refreshing task list after creates...")
        remote_tasks = client.list_all_tasks(PROJECT_ID)
        issue_by_roadmap = parse_roadmap_ids(remote_tasks)
        github_by_task = build_github_map(remote_tasks)
        tasks_by_id = index_tasks_by_kaneo_id(remote_tasks)
        epic_kaneo_id, task_kaneo_id = populate_id_maps(plan, issue_by_roadmap)

    relations_created = 0
    relations_skipped = 0
    print("Wiring subtask relations (epic -> leaf)...")
    for epic in plan:
        parent_id = epic_kaneo_id.get(epic.epic_id)
        if not parent_id:
            failures.append(f"subtask missing epic parent: {epic.epic_id}")
            continue
        for task in epic.tasks:
            child_id = task_kaneo_id.get(task.task_id)
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
            child_id = task_kaneo_id.get(task.task_id)
            if not child_id:
                continue
            for dep in task.depends_on:
                blocker_id = task_kaneo_id.get(dep) or issue_by_roadmap.get(dep)
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
            child_id = task_kaneo_id.get(task.task_id)
            if not child_id or not task.depends_on:
                continue
            current = tasks_by_id.get(child_id) or client.get_task(child_id)
            description = current.get("description", "")
            updated = patch_depends_section(description, task.depends_on, issue_by_roadmap, github_by_task)
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
        parent_id = epic_kaneo_id.get(epic.epic_id)
        if not parent_id:
            continue
        child_rows: list[tuple[PlannedTask, str | None]] = []
        for task in epic.tasks:
            child_id = task_kaneo_id.get(task.task_id)
            issue_num = github_by_task.get(child_id or "") if child_id else None
            child_rows.append((task, issue_num))
        final_desc = build_epic_description_final(epic, child_rows)
        current = tasks_by_id.get(parent_id) or client.get_task(parent_id)
        if current.get("description") == final_desc:
            continue
        try:
            client.update_description(parent_id, final_desc)
            epics_patched += 1
        except Exception as exc:
            failures.append(f"patch epic {epic.epic_id}: {exc}")

    labels_attached = 0
    print("Attaching per-task labels (epics get epic + phase; leaves get phase + task labels)...")
    for epic in plan:
        parent_id = epic_kaneo_id.get(epic.epic_id)
        if parent_id:
            current = tasks_by_id.get(parent_id) or client.get_task(parent_id)
            for label_name in {EPIC_LABEL, epic.phase_id}:
                try:
                    if ensure_task_label(client, current, label_name):
                        labels_attached += 1
                        current.setdefault("labels", []).append({"name": label_name})
                except Exception as exc:
                    failures.append(f"label {label_name} on epic {epic.epic_id}: {exc}")

        for task in epic.tasks:
            child_id = task_kaneo_id.get(task.task_id)
            if not child_id:
                continue
            current = tasks_by_id.get(child_id) or client.get_task(child_id)
            attach_names = set(task.labels) | {task.phase_id}
            if task.external_dependency:
                attach_names.add("external-dependency")
            for label_name in attach_names:
                try:
                    if ensure_task_label(client, current, label_name):
                        labels_attached += 1
                        current.setdefault("labels", []).append({"name": label_name})
                except Exception as exc:
                    failures.append(f"label {label_name} on {task.task_id}: {exc}")

    status_updated = 0
    print(f"Setting status -> {READY_STATUS} where not already...")
    all_kaneo_ids = list({*task_kaneo_id.values(), *epic_kaneo_id.values()})
    for kaneo_id in all_kaneo_ids:
        current = tasks_by_id.get(kaneo_id) or client.get_task(kaneo_id)
        if current.get("status") == READY_STATUS:
            continue
        try:
            client.update_status(kaneo_id, READY_STATUS)
            status_updated += 1
        except Exception as exc:
            failures.append(f"status {kaneo_id}: {exc}")

    print("\n=== APPLY SUMMARY ===")
    print(f"Remote tasks: {len(remote_tasks)}")
    print(f"Created epics: {created_epics} (pre-existing epics: {skipped_epics})")
    print(f"Created leaf tasks: {created_tasks} (pre-existing leaves: {skipped_tasks})")
    print(f"Relations: {relations_created} created, {relations_skipped} already existed")
    print(f"Descriptions patched: {patched} leaves, {epics_patched} epics")
    print(f"Labels attached: {labels_attached}")
    print(f"Status moved to {READY_STATUS}: {status_updated}")
    print(f"Failures: {len(failures)}")
    for failure in failures[:20]:
        print(f"  - {failure}")
    if len(failures) > 20:
        print(f"  ... and {len(failures) - 20} more")


def main() -> int:
    load_env_local()
    parser = argparse.ArgumentParser(description="Create Kaneo issues from roadmap.yaml")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--dry-run", action="store_true", help="Print planned creates without API writes")
    group.add_argument("--apply", action="store_true", help="Create/finalize issues in Kaneo")
    parser.add_argument("--base-url", default=os.environ.get("KANEO_BASE_URL", DEFAULT_BASE_URL))
    args = parser.parse_args()

    api_key = os.environ.get("KANEO_API_KEY")
    if not api_key:
        print("ERROR: KANEO_API_KEY is required (env or .env.local)", file=sys.stderr)
        return 1

    roadmap = yaml.safe_load(ROADMAP_PATH.read_text(encoding="utf-8"))
    plan = plan_from_roadmap(roadmap)

    client = KaneoClient(args.base_url, api_key)
    remote_tasks = client.list_all_tasks(PROJECT_ID)
    existing = parse_roadmap_ids(remote_tasks)

    if args.dry_run:
        print_dry_run(plan, existing, len(remote_tasks))
        return 0

    user_id = os.environ.get("KANEO_USER_ID", DEFAULT_USER_ID)
    apply_plan(client, plan, user_id)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
