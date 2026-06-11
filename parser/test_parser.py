import unittest
import ast
from parser.parser import SymbolTable, extract_models, resolve_imports


class TestDataclassExtraction(unittest.TestCase):
    def test_dataclass_detected(self):
        source_code = """
from dataclasses import dataclass

@dataclass
class User:
    name: str
    age: int
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["name"]["type"], "string")
        self.assertEqual(models["User"]["fields"]["age"]["type"], "int")

    def test_dataclass_empty(self):
        source_code = """
from dataclasses import dataclass

@dataclass
class Empty:
    pass
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("Empty", models)
        self.assertEqual(len(models["Empty"]["fields"]), 0)


class TestTypedDictExtraction(unittest.TestCase):
    def test_typeddict_detected(self):
        source_code = """
from typing import TypedDict

class User(TypedDict):
    name: str
    age: int
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["name"]["type"], "string")

    def test_typeddict_optional_field(self):
        source_code = """
from typing import TypedDict, Optional

class User(TypedDict):
    email: Optional[str]
    age: int
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["email"]["type"], "optional(string)")


class TestNamedTupleExtraction(unittest.TestCase):
    def test_namedtuple_detected(self):
        source_code = """
from typing import NamedTuple

class User(NamedTuple):
    name: str
    age: int
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["name"]["type"], "string")


class TestAnnotatedTypeResolution(unittest.TestCase):
    def test_annotated_unwraps_to_inner_type(self):
        source_code = """
from typing import Annotated
from pydantic import BaseModel, Field

class User(BaseModel):
    name: Annotated[str, Field(min_length=1)]
    age: Annotated[int, Field(ge=0)]
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["name"]["type"], "string")
        self.assertEqual(models["User"]["fields"]["age"]["type"], "int")

    def test_annotated_single_arg(self):
        source_code = """
from typing import Annotated
from pydantic import BaseModel

class Item(BaseModel):
    price: Annotated[float, ...]
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertEqual(models["Item"]["fields"]["price"]["type"], "float")


class TestMappedColumnExtraction(unittest.TestCase):
    def test_mapped_column_with_primary_key(self):
        source_code = """
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column

class Base(DeclarativeBase):
    pass

class User(Base):
    __tablename__ = 'users'
    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(50))
    email: Mapped[str] = mapped_column(String(100), nullable=False)
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["id"]["type"], "int")
        self.assertEqual(models["User"]["fields"]["id"]["nullable"], False)
        self.assertEqual(models["User"]["fields"]["name"]["type"], "string")
        self.assertEqual(models["User"]["fields"]["email"]["nullable"], False)

    def test_mapped_column_single_model(self):
        source_code = """
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column

class Base(DeclarativeBase):
    pass

class Product(Base):
    __tablename__ = 'products'
    sku: Mapped[str] = mapped_column(primary_key=True)
    price: Mapped[float]
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("Product", models)
        self.assertEqual(models["Product"]["fields"]["sku"]["type"], "string")
        self.assertEqual(models["Product"]["fields"]["sku"]["nullable"], False)
        self.assertEqual(models["Product"]["fields"]["price"]["type"], "float")


class TestInheritedFields(unittest.TestCase):
    def test_inherited_fields_merged(self):
        source_code = """
from pydantic import BaseModel

class UserBase(BaseModel):
    id: int
    created_at: str

class User(UserBase):
    name: str
    email: str
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertIn("id", models["User"]["fields"])
        self.assertIn("created_at", models["User"]["fields"])
        self.assertIn("name", models["User"]["fields"])
        self.assertEqual(models["User"]["fields"]["id"]["type"], "int")
        self.assertEqual(models["User"]["fields"]["name"]["type"], "string")

    def test_inherited_fields_do_not_override(self):
        source_code = """
from pydantic import BaseModel

class BaseWithId(BaseModel):
    id: int

class User(BaseWithId):
    id: str
    name: str
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        # Child's own definition should win
        self.assertEqual(models["User"]["fields"]["id"]["type"], "string")

    def test_multi_level_inheritance(self):
        source_code = """
from pydantic import BaseModel

class TimestampMixin(BaseModel):
    created_at: str
    updated_at: str

class UserBase(TimestampMixin):
    id: int

class User(UserBase):
    name: str
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertIn("created_at", models["User"]["fields"])
        self.assertIn("updated_at", models["User"]["fields"])
        self.assertIn("id", models["User"]["fields"])
        self.assertIn("name", models["User"]["fields"])


class TestPythonParser(unittest.TestCase):
    def test_sqlalchemy_model_extraction(self):
        source_code = """
from sqlalchemy import Column, Integer, String
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()

class User(Base):
    __tablename__ = 'users'
    id = Column(Integer, primary_key=True)
    username = Column(String(50), nullable=False)
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        # Integer gets normalized to "int" by TYPE_NORMALIZE_MAP
        self.assertEqual(models["User"]["fields"]["id"]["type"], "int")
        # primary_key=True implies nullable=False
        self.assertEqual(models["User"]["fields"]["id"]["nullable"], False)
        self.assertEqual(models["User"]["fields"]["username"]["nullable"], False)

    def test_import_alias_resolution(self):
        source_code = """
import sqlalchemy as sa
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()

class User(Base):
    __tablename__ = 'users'
    id = sa.Column(sa.Integer, primary_key=True)
    username = sa.Column(sa.String(50), nullable=False)
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["id"]["type"], "int")

    def test_wildcard_import_graceful_fallback(self):
        """Wildcard imports are ignored; symbols resolve to 'unresolved_type'."""
        source_code = """
from sqlalchemy import *
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()

class User(Base):
    __tablename__ = 'users'
    id = Column(Integer, primary_key=True)
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        # Column/Integer unresolved via wildcard — type falls back gracefully
        self.assertIn(models["User"]["fields"]["id"]["type"],
                      ["int", "unresolved_type"])

    def test_pydantic_model_extraction(self):
        source_code = """
from pydantic import BaseModel

class Item(BaseModel):
    name: str
    price: float
    in_stock: bool = True
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("Item", models)
        self.assertEqual(models["Item"]["fields"]["name"]["type"], "string")
        self.assertEqual(models["Item"]["fields"]["price"]["type"], "float")
        self.assertEqual(models["Item"]["fields"]["in_stock"]["type"], "bool")

    def test_optional_type_resolution(self):
        source_code = """
from typing import Optional
from pydantic import BaseModel

class User(BaseModel):
    email: Optional[str] = None
    age: Optional[int] = None
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["email"]["type"], "optional(string)")
        self.assertTrue(models["User"]["fields"]["email"]["nullable"])
        self.assertEqual(models["User"]["fields"]["age"]["type"], "optional(int)")

    def test_empty_file(self):
        tree = ast.parse("")
        models = extract_models(tree)
        self.assertEqual(models, {})

    def test_no_model_classes(self):
        source_code = """
def helper():
    pass

SOME_CONSTANT = 42
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertEqual(models, {})


if __name__ == '__main__':
    unittest.main()
