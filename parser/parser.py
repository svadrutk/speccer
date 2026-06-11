import ast
import json
import sys
import os

TYPE_NORMALIZE_MAP = {
    "Integer": "int",
    "String": "string",
    "Text": "text",
    "Boolean": "bool",
    "Float": "float",
    "DateTime": "datetime",
    "UUID": "uuid",
    "int": "int",
    "str": "string",
    "bool": "bool",
    "float": "float",
}


class SymbolTable:
    """Cross-file symbol registry for static import resolution."""
    def __init__(self):
        self.symbols = {}
        self.aliases = {}
        self.files = {}

    def add_file(self, file_path, models):
        self.files[file_path] = models
        for name, model in models.items():
            self.symbols[name] = {"source_file": file_path, "fields": model["fields"]}

    def resolve_import(self, node, current_file):
        """Resolve an import node and return dict of {name: source_file} mappings."""
        resolved = {}
        if isinstance(node, ast.Import):
            for alias in node.names:
                name = alias.asname or alias.name
                self.aliases[name] = alias.name
        elif isinstance(node, ast.ImportFrom):
            if node.names and node.names[0].name == '*':
                return None
            module = node.module or ""
            base_dir = os.path.dirname(current_file)
            level = node.level or 0
            if level > 0:
                parts = module.split(".")
                for _ in range(level - 1):
                    base_dir = os.path.dirname(base_dir)
                for fpath in self.files:
                    rel = os.path.relpath(fpath, base_dir)
                    rel_no_ext = rel.replace(".py", "").replace(os.sep, ".")
                    if rel_no_ext == module or rel_no_ext.endswith("." + module):
                        for alias in node.names:
                            name = alias.asname or alias.name
                            resolved[name] = fpath
            else:
                parts = module.split(".")
                for fpath in self.files:
                    rel = fpath.replace(".py", "").replace(os.sep, ".")
                    if rel == module or rel.endswith("." + module):
                        for alias in node.names:
                            name = alias.asname or alias.name
                            resolved[name] = fpath
        return resolved or None


def resolve_imports(node, current_file, file_registry=None):
    """Standalone convenience wrapper around SymbolTable.resolve_import."""
    symtab = SymbolTable()
    if file_registry:
        for path, models in file_registry.items():
            symtab.add_file(path, models)
    return symtab.resolve_import(node, current_file)


def normalize_type(type_node):
    if isinstance(type_node, ast.Name):
        raw = type_node.id
        return TYPE_NORMALIZE_MAP.get(raw, raw)
    elif isinstance(type_node, ast.Attribute):
        raw = type_node.attr
        return TYPE_NORMALIZE_MAP.get(raw, raw)
    elif isinstance(type_node, ast.Subscript):
        value = normalize_type(type_node.value)
        if value == "Annotated":
            if isinstance(type_node.slice, ast.Tuple) and type_node.slice.elts:
                return normalize_type(type_node.slice.elts[0])
            return normalize_type(type_node.slice)
        if value == "Mapped":
            return normalize_type(type_node.slice)
        slice_val = normalize_type(type_node.slice)
        if value in ("Optional", "Union"):
            return f"optional({slice_val})"
        return f"{value}[{slice_val}]"
    elif isinstance(type_node, ast.Constant):
        if type_node.value is None:
            return "None"
        return str(type_node.value)
    elif isinstance(type_node, ast.BinOp) and isinstance(type_node.op, ast.BitOr):
        left = normalize_type(type_node.left)
        right = normalize_type(type_node.right)
        if right == "None":
            return f"optional({left})"
        return f"union({left}, {right})"
    elif isinstance(type_node, ast.Tuple):
        parts = [normalize_type(elt) for elt in type_node.elts]
        return ", ".join(parts)
    return "unresolved_type"


MODEL_BASE_NAMES = frozenset((
    "Base", "SQLModel", "Model", "BaseModel",
    "TypedDict", "NamedTuple",
))


def _is_dataclass(node):
    for decorator in node.decorator_list:
        if isinstance(decorator, ast.Name) and decorator.id == "dataclass":
            return True
        if isinstance(decorator, ast.Attribute) and decorator.attr == "dataclass":
            return True
    return False


def _merge_inherited_fields(node, fields, models):
    for base in node.bases:
        parent_name = normalize_type(base)
        if parent_name in models:
            for fname, fdata in models[parent_name]["fields"].items():
                if fname not in fields:
                    fields[fname] = dict(fdata)


def extract_models(tree, file_path=""):
    models = {}
    for node in ast.walk(tree):
        if not isinstance(node, ast.ClassDef):
            continue

        is_model = False
        for base in node.bases:
            base_name = normalize_type(base)
            if base_name in MODEL_BASE_NAMES:
                is_model = True
                break

        if not is_model and _is_dataclass(node):
            is_model = True

        if not is_model:
            for base in node.bases:
                base_name = normalize_type(base)
                if base_name in models:
                    is_model = True
                    break

        if not is_model:
            continue

        fields = {}
        _merge_inherited_fields(node, fields, models)

        for item in node.body:
            if isinstance(item, ast.Assign):
                for target in item.targets:
                    if isinstance(target, ast.Name):
                        f_name = target.id
                        f_type = "unresolved_type"
                        nullable = True
                        if isinstance(item.value, ast.Call):
                            func_name = normalize_type(item.value.func)
                            if func_name == "Column":
                                for arg in item.value.args:
                                    f_type = normalize_type(arg)
                                has_primary_key = False
                                for kw in item.value.keywords:
                                    if kw.arg == "nullable" and isinstance(kw.value, ast.Constant):
                                        nullable = kw.value.value
                                    if kw.arg == "primary_key" and isinstance(kw.value, ast.Constant):
                                        has_primary_key = kw.value.value
                                if has_primary_key:
                                    nullable = False
                        fields[f_name] = {"type": f_type, "nullable": nullable}
            elif isinstance(item, ast.AnnAssign):
                if isinstance(item.target, ast.Name):
                    f_name = item.target.id
                    f_type = normalize_type(item.annotation)
                    nullable = "optional" in f_type
                    if isinstance(item.value, ast.Call):
                        func_name = normalize_type(item.value.func)
                        if func_name == "mapped_column":
                            has_primary_key = False
                            for kw in item.value.keywords:
                                if kw.arg == "nullable" and isinstance(kw.value, ast.Constant):
                                    nullable = kw.value.value
                                if kw.arg == "primary_key" and isinstance(kw.value, ast.Constant):
                                    has_primary_key = kw.value.value
                            if has_primary_key:
                                nullable = False
                    fields[f_name] = {"type": f_type, "nullable": nullable}

        models[node.name] = {
            "name": node.name,
            "fields": fields
        }
    return models


def main():
    if len(sys.argv) < 2:
        print("Usage: parser.py <file1.py> ...", file=sys.stderr)
        sys.exit(1)

    symtab = SymbolTable()
    results = {}

    for f_path in sys.argv[1:]:
        if os.path.exists(f_path):
            with open(f_path, "r", encoding="utf-8") as f:
                source = f.read()
            tree = ast.parse(source, filename=f_path)
            models = extract_models(tree, f_path)
            symtab.add_file(f_path, models)
            results.update(models)

    for f_path in sys.argv[1:]:
        if os.path.exists(f_path):
            with open(f_path, "r", encoding="utf-8") as f:
                source = f.read()
            tree = ast.parse(source, filename=f_path)
            for node in ast.iter_child_nodes(tree):
                if isinstance(node, (ast.Import, ast.ImportFrom)):
                    symtab.resolve_import(node, f_path)

    print("===SPECCER-JSON-START===")
    print(json.dumps(results))


if __name__ == '__main__':
    main()
